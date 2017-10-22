package main

import (
	"fmt"
	"html/template"
	"labix.org/v2/mgo/bson"
	"mvdan.cc/xurls"
	"strconv"
	"strings"
	"time"
)

const (
	cRed    = "#ffb3b3"
	cYellow = "#ffff80"
	cGreen  = "#8cf28c"
	cGray   = "#262626"
)

var urlsRe = xurls.Strict()

// Helper to parse elo/score in old format and rewrite in new formatting
func parse_old_elo(s string) string {

	var llr, llrMax, a, b, x float64
	var g, t int
	var h string

	fmt.Sscanf(s, "%4s %f (%f,%f) [%f,%f]", &h, &llr, &x, &llrMax, &a, &b)

	if h == "LLR:" { // SPRT
		p := int(llr * 100 / (llrMax + 0.01))
		return fmt.Sprintf("LLR: %d%% SPRT[%d, %d]", p, int(a), int(b))
	}
	fmt.Sscanf(s, "%d/%d %10s", &g, &t, &h)

	if h == "iterations" { // SPSA
		p := int(g * 100 / (t + 1))
		return fmt.Sprintf("SPSA: %d/%d (%d%%)", g, t, p)
	}
	fmt.Sscanf(s, "%4s %f +-%f (%d%%) LOS: %f", &h, &a, &b, &g, &x)

	if h == "ELO:" { // Fixed games
		return fmt.Sprintf("ELO: %.2f +-%.2f LOS: %.2f%%", a, b, x)
	}
	return s
}

type FmtFunc struct{}

// Set the led color according to the test state. Return a map to workaround
// the single value limit of the template functions.
func (_ FmtFunc) Led(finished bool, tasks []interface{}) bson.M {

	if finished {
		return bson.M{"Color": "gray", "Workers": ""}
	}
	workers := 0
	for _, t := range tasks {
		if t.(bson.M)["active"].(bool) {
			workers++
		}
	}
	if workers > 0 {
		return bson.M{"Color": "limegreen", "Workers": strconv.Itoa(workers)}
	}
	return bson.M{"Color": "gold", "Workers": ""}
}

// Compute ELO and SPRT stats of the test
func (_ FmtFunc) Elo(results bson.M, tc string, threads int, sprt, spsa, results_info interface{}) bson.M {

	colorMap := map[string](string){"#FF6A6A": cRed, "yellow": cYellow, "#44EB44": cGreen}
	var info, crashes, color, border string

	if strings.HasPrefix(tc, "60") { // LTC
		border = cGray
	}

	w, l, d := results["wins"].(int), results["losses"].(int), results["draws"].(int)
	c, ok1 := results["crashes"].(int) // New tests don't have this info
	t, ok2 := results["time_losses"].(int)

	if ok1 && ok2 {
		crashes = fmt.Sprintf("(c%v t%v)", c, t)
	}
	// For finished runs results are saved in results_info.info that is a
	// slice of strings, usually 2, one for each box line.
	if results_info != nil {
		r := results_info.(bson.M)
		i, ok := r["info"].([]interface{})
		if ok && len(r) > 0 {
			info = parse_old_elo(i[0].(string)) // Only first line is used
			color, _ = colorMap[r["style"].(string)]
		}
	} else if sprt != nil {
		s := sprt.(bson.M)
		elo0 := s["elo0"].(float64)
		alpha := s["alpha"].(float64)
		elo1 := s["elo1"].(float64)
		beta := s["beta"].(float64)
		sprt := Compute_sprt(w, l, d, elo0, alpha, elo1, beta)
		p := int(sprt.llr * 100 / (sprt.upper_bound + 0.0001))
		info = fmt.Sprintf("LLR: %d%% SPRT[%d, %d]", p, int(elo0), int(elo1))
	} else if spsa != nil {
		s := spsa.(bson.M)
		i := s["iter"].(int)
		n := s["num_iter"].(int)
		p := i * 100 / n
		info = fmt.Sprintf("SPSA: %d/%d (%d%%)", i, n, p)
	} else {
		el, elo95, los := Compute_elo(w, l, d)
		info = fmt.Sprintf("ELO: %.2f +-%.2f LOS: %.2f%%", el, elo95, los)
	}
	s := "%s tc %s th %v\nTot: %v W: %v L: %v D: %v %s"
	info = fmt.Sprintf(s, info, tc, threads, w+l+d, w, l, d, crashes)
	return bson.M{"Color": color, "Border": border, "Info": info}
}

// Fancy formatting of time since test has been submitted
func (_ FmtFunc) Date(start_time time.Time) string {

	d := time.Since(start_time)
	m, h := d.Minutes(), d.Hours()

	if m < 1.0 {
		return "now"
	} else if m < 60.0 {
		return fmt.Sprintf("%v min", int(m))
	} else if int(h) == 1 {
		return fmt.Sprintf("%v hour", int(h))
	} else if h < 24.0 {
		return fmt.Sprintf("%v hours", int(h))
	} else if int(h/24) == 1 {
		return fmt.Sprintf("%v day", int(h/24))
	} else if h < 24.0*3 {
		return fmt.Sprintf("%v days", int(h/24))
	}
	return start_time.Format("02-01-2006")
}

// Convert any url in a string in a href. Return a template.HTML
// to avoid template engine escapes the string.
func (_ FmtFunc) UnescapeURL(in string) template.HTML {

	list := urlsRe.FindAllString(in, -1)
	if len(list) == 0 {
		return template.HTML(in)
	}
	out := in
	for _, u := range list {
		s := fmt.Sprintf("<a href=\"%s\" target=\"_blank\">%s</a>", u, u)
		out = strings.Replace(out, u, s, -1)
	}
	return template.HTML(out)
}
