package main

import (
	"math"
)

func erf(x float64) float64 {
	//Go defines math.Erf(), but we keep compatibility with fishtest version
	a := 8 * (math.Pi - 3) / (3 * math.Pi * (4 - math.Pi))
	x2 := x * x
	y := -x2 * (4/math.Pi + a*x2) / (1 + a*x2)
	return math.Copysign(math.Sqrt(1-math.Exp(y)), x)
}

func erf_inv(x float64) float64 {
	// Above erf formula inverted analytically
	a := 8 * (math.Pi - 3) / (3 * math.Pi * (4 - math.Pi))
	y := math.Log(1 - x*x)
	z := 2/(math.Pi*a) + y/2
	return math.Copysign(math.Sqrt(math.Sqrt(z*z-y/a)-z), x)
}

func phi(q float64) float64 {
	// Cumlative distribution function for the standard Gaussian law: quantile -> probability
	return 0.5 * (1 + erf(q/math.Sqrt(2)))
}

func phi_inv(p float64) float64 {
	// Quantile function for the standard Gaussian law: probability -> quantile
	return math.Sqrt(2) * erf_inv(2*p-1)
}

func elo(x float64) float64 {
	if x <= 0 {
		return 0.0
	}
	return -400 * math.Log10(1/x-1)
}

func bayeselo_to_proba(elo, drawelo float64) (float64, float64, float64) {
	// elo is expressed in BayesELO (relative to the choice drawelo).
	// Returns a probability, pwin, ploss, pdraw
	pwin := 1.0 / (1.0 + math.Pow(10.0, (-elo+drawelo)/400.0))
	ploss := 1.0 / (1.0 + math.Pow(10.0, (elo+drawelo)/400.0))
	pdraw := 1.0 - pwin - ploss
	return pwin, ploss, pdraw
}

func proba_to_bayeselo(pwin, ploss float64) (float64, float64) {
	// Takes a probability: pwin, ploss
	// Returns elo, drawelo
	elo := 200.0 * math.Log10(pwin/ploss*(1-ploss)/(1-pwin))
	drawelo := 200.0 * math.Log10((1-ploss)/ploss*(1-pwin)/pwin)
	return elo, drawelo
}

type SPRT struct {
	finished    bool
	state       string
	llr         float64
	lower_bound float64
	upper_bound float64
}

func Compute_elo(w, l, d int) (float64, float64, float64) {
	// win/loss/draw ratio
	n := float64(w + l + d)
	pw, pl, pd := float64(w)/n, float64(l)/n, float64(d)/n

	// mu is the empirical mean of the variables (Xi), assumed i.i.d.
	mu := pw + pd/2

	// stdev is the empirical standard deviation of the random variable (X1+...+X_N)/N
	stdev := math.Sqrt(pw*math.Pow((1-mu), 2)+pl*math.Pow((0-mu), 2)+pd*math.Pow((0.5-mu), 2)) / math.Sqrt(n)

	// 95% confidence interval for mu
	mu_min := mu + phi_inv(0.025)*stdev
	mu_max := mu + phi_inv(0.975)*stdev

	el := elo(mu)
	elo95 := (elo(mu_max) - elo(mu_min)) / 2
	los := phi((mu - 0.5) / stdev)
	return el, elo95, los
}

/*
  Sequential Probability Ratio Test
  H0: elo = elo0
  H1: elo = elo1
  alpha = max typeI error (reached on elo = elo0)
  beta = max typeII error for elo >= elo1 (reached on elo = elo1)
  w, l, d are the number of wins, losses and draws

  Returns a SPRT struct:
  finished - bool, true means test is finished, false means continue sampling
  state - string, 'accepted', 'rejected' or ''
  llr - Log-likelihood ratio
  lower_bound/upper_bound - SPRT bounds
*/
func Compute_sprt(w, l, d int, elo0, alpha, elo1, beta float64) SPRT {

	ww, ll, dd := float64(w), float64(l), float64(d)
	result := SPRT{false, "", 0.0, math.Log(beta / (1 - alpha)), math.Log((1 - beta) / alpha)}

	if w == 0 || l == 0 || d == 0 {
		return result
	}
	// Estimate drawelo out of sample
	n := ww + ll + dd
	_, drawelo := proba_to_bayeselo(ww/n, ll/n)

	// Probability laws under H0 and H1
	p0w, p0l, p0d := bayeselo_to_proba(elo0, drawelo)
	p1w, p1l, p1d := bayeselo_to_proba(elo1, drawelo)

	// Log-Likelyhood Ratio
	result.llr = ww*math.Log(p1w/p0w) + ll*math.Log(p1l/p0l) + dd*math.Log(p1d/p0d)

	if result.llr < result.lower_bound {
		result.finished = true
		result.state = "rejected"
	} else if result.llr > result.upper_bound {
		result.finished = true
		result.state = "accepted"
	}
	return result
}
