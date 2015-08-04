// Copyright 2012 Dorival de Moraes Pedroso. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bytes"
	"math"
	"time"

	"github.com/cpmech/goga"
	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/io"
	"github.com/cpmech/gosl/la"
	"github.com/cpmech/gosl/plt"
	"github.com/cpmech/gosl/rnd"
	"github.com/cpmech/gosl/utl"
)

func main() {

	// Problems 1 to 5 have all variables as standard variables => μ=0 and σ=1 => y = x
	// References
	//  [1] Santos SR, Matioli LC and Beck AT. New optimization algorithms for structural
	//      reliability analysis. Computer Modeling in Engineering & Sciences, 83(1):23-56; 2012
	//      doi:10.3970/cmes.2012.083.023
	//  [2] Borri A and Speranzini E. Structural reliability analysis using a standard deterministic
	//      finite element code. Structural Safety, 19(4):361-382; 1997
	//      doi:10.1016/S0167-4730(97)00017-9
	//  [3] Grooteman F.  Adaptive radial-based importance sampling method or structural
	//      reliability. Structural safety, 30:533-542; 2008
	//      doi:10.1016/j.strusafe.2007.10.002
	//  [4] Wang L and Grandhi RV. Higher-order failure probability calculation using nonlinear
	//      approximations. Computer Methods in Applied Mechanics and Engineering, 168(1-4):185-206;
	//      1999 doi:10.1016/S0045-7825(98)00140-6
	//  [5] Santosh TV, Saraf RK, Ghosh AK and KushwahaHS. Optimum step length selection rule in
	//      modified HL–RF method for structural reliability. International Journal of Pressure
	//      Vessels and Piping, 83(10):742-748; 2006 doi:10.1016/j.ijpvp.2006.07.004
	//  [6] Haldar and Mahadevan. Probability, reliability and statistical methods in engineering
	//      and design. John Wiley & Sons. 304p; 2000.

	// catch errors
	defer func() {
		if err := recover(); err != nil {
			io.PfRed("ERROR: %v\n", err)
		}
	}()

	// read parameters
	fn := "rel-prob1to5"
	fn, _ = io.ArgToFilename(0, fn, ".json", true)
	C := goga.ReadConfParams(fn)
	io.Pf("\n%s\nproblem # %v\n", utl.PrintThickLine(80), C.Problem)

	// initialise random numbers generator
	rnd.Init(C.Seed)

	// problems's data
	ϵ := 1e-2                       // tolerance for strategy 2
	npts := 101                     // number of points for contour plot
	var g func(x []float64) float64 // limit state function
	var βref float64                // reference β (if available)
	var xref []float64              // reference x (if available)
	var μ, σ []float64              // mean and std deviation (if not standard)
	var ds []string                 // distribution. <nil> means standard with μ=0, σ=1
	vmin := []float64{-4, -4}       // x or y min. default is standard vars; thus x==y
	vmax := []float64{4, 4}         // x or y max. default is standard vars; thus x==y
	axequal := true                 // plot with axis.equal

	// set problem
	switch C.Problem {

	// problem # 1 of [1] and Eq. (A.5) of [2]
	case 1:
		g = func(x []float64) float64 {
			return 0.1*math.Pow(x[0]-x[1], 2) - (x[0]+x[1])/math.Sqrt2 + 2.5
		}
		βref = 2.5 // from [1]
		xref = []float64{1.7677, 1.7677}
		μ = []float64{0, 0}
		σ = []float64{1, 1}
		ds = []string{"nrm", "nrm"}

	// problem # 2 of [1] and Eq. (A.6) of [2]
	case 2:
		g = func(x []float64) float64 {
			return -0.5*math.Pow(x[0]-x[1], 2) - (x[0]+x[1])/math.Sqrt2 + 3
		}
		βref = 1.658 // from [2]
		xref = []float64{-0.7583, 1.4752}

	// problem # 3 from [1] and # 6 from [3]
	case 3:
		g = func(x []float64) float64 {
			return 2 - x[1] - 0.1*math.Pow(x[0], 2) + 0.06*math.Pow(x[0], 3)
		}
		βref = 2 // from [1]
		xref = []float64{0, 2}

	// problem # 4 from [1] and # 8 from [3]
	case 4:
		g = func(x []float64) float64 {
			return 3 - x[1] + 256*math.Pow(x[0], 4)
		}
		npts = 101
		βref = 3 // from [1]
		xref = []float64{0, 3}

	// problem # 5 from [1] and # 1 from [4] (modified)
	case 5:
		shift := 0.1
		g = func(x []float64) float64 {
			return 1 + math.Pow(x[0]+x[1]+shift, 2)/4 - 4*math.Pow(x[0]-x[1]+shift, 2)
		}
		βref = 0.3536 // from [1]
		xref = []float64{-βref * math.Sqrt2 / 2, βref * math.Sqrt2 / 2}

	// problem # 7 from [1] and example # 1 (case1) from [5]
	// x1 and x2 are normally distributed and statistically independent with
	case 6:
		g = func(x []float64) float64 {
			return math.Pow(x[0], 3) + math.Pow(x[1], 3) - 18
		}
		ϵ = 0.1
		βref = 2.2401 // from [1]
		μ = []float64{10, 10}
		σ = []float64{5, 5}
		ds = []string{"nrm", "nrm"}

	// problem # 8 from [1] and example # 1 (case2) from [5]
	// x1 and x2 are normally distributed and statistically independent with
	case 7:
		g = func(x []float64) float64 {
			return math.Pow(x[0], 3) + math.Pow(x[1], 3) - 18
		}
		ϵ = 0.1
		βref = 2.2260 // from [1]
		μ = []float64{10, 9.9}
		σ = []float64{5, 5}
		ds = []string{"nrm", "nrm"}

	// problem # 9 from [1] and case 7 of [3]
	// x1 and x2 are normally distributed and statistically independent with
	case 8:
		g = func(x []float64) float64 {
			return 2.5 - 0.2357*(x[0]-x[1]) + 0.0046*math.Pow(x[0]+x[1]-20, 4)
		}
		βref = 2.5 // from [1]
		μ = []float64{10, 10}
		σ = []float64{3, 3}
		ds = []string{"nrm", "nrm"}

	// problem # 10 from [1] and example # 2 from [5]
	// x1 and x2 are normally distributed and statistically independent with
	case 9:
		g = func(x []float64) float64 {
			return math.Pow(x[0], 3) + math.Pow(x[1], 3) - 67.5
		}
		βref = 1.9003 // from [1]
		μ = []float64{10, 10}
		σ = []float64{5, 5}
		ds = []string{"nrm", "nrm"}

	// problem # 11 from [1] and case 2 of [3]
	// x1 and x2 are normally distributed and statistically independent with
	case 10:
		g = func(x []float64) float64 {
			return x[0]*x[1] - 146.14
		}
		βref = 5.4280 // from [1]
		μ = []float64{78064.4, 0.0104}
		σ = []float64{11709.7, 0.00156}
		ds = []string{"nrm", "nrm"}
		axequal = false
		vmin = []float64{-6, -6} // x or y min. default is standard vars; thus x==y
		vmax = []float64{6, 6}   // x or y max. default is standard vars; thus x==y

	// problem # 12 from [1] and example 2 of [4]
	// x1 and x2 are normally distributed and statistically independent with
	case 11:
		g = func(x []float64) float64 {
			return 2.2257 - 0.025*math.Sqrt2*math.Pow(x[0]+x[1]-20, 3)/27 + 0.2357*(x[0]-x[1])
		}
		//ϵ = 0.5
		βref = 2.2257 // from [1]
		μ = []float64{10, 10}
		σ = []float64{3, 3}
		ds = []string{"nrm", "nrm"}

	// problem # 14 from [1]
	// x1 and x2 are log-normally distributed and statistically independent with
	case 12:
		g = func(x []float64) float64 {
			return x[0]*x[1] - 1140
		}
		βref = 5.2127 // from [1]
		μ = []float64{38, 54}
		σ = []float64{3.8, 2.7}
		ds = []string{"log", "log"}

	default:
		chk.Panic("problem number %d is invalid", C.Problem)
	}

	// use original variables
	if C.Strategy < 3 && μ != nil {
		vmin, vmax = make([]float64, 2), make([]float64, 2)
		for i := 0; i < 2; i++ {
			vmin[i] = μ[i] - 2*μ[i]
			vmax[i] = μ[i] + 2*μ[i]
		}
	}

	// objective value function
	ovfunc := func(ind *goga.Individual, idIsland, t int, report *bytes.Buffer) (ova, oor float64) {

		switch C.Strategy {

		// argmin_x{ β(y(x)) | g(x) ≤ 0 }
		case 1:
			x := ind.GetFloats()             // must be inside ovfunc to avoid data race problems
			y := calc_norm_vars(ds, μ, σ, x) // original => standard normal variables
			b := la.VecDot(y, y)             // squared distance from origin in normalised space
			ova = b                          // ova ← y dot y
			oor = utl.GtePenalty(0, g(x), 1) // oor ← 0 ≥ g(x)

		// argmin_x{ β(y(x)) + c(x) | |g(x)| ≤ ϵ }
		case 2:
			x := ind.GetFloats()                      // must be inside ovfunc to avoid data race problems
			y := calc_norm_vars(ds, μ, σ, x)          // original => standard normal variables
			b := la.VecDot(y, y)                      // squared distance from origin in normalised space
			c := utl.GtePenalty(ϵ, math.Abs(g(x)), 1) // c ← ϵ ≥ |g(x)|
			ova = b + c                               // ova ← y dot y + c
			oor = c                                   // oor ← c

		// argmin_y{ β(y) | g(x(y)) ≤ 0 }
		case 3:
			y := ind.GetFloats()             // must be inside ovfunc to avoid data race problems
			x := calc_orig_vars(ds, μ, σ, y) // standard normal variables => original
			b := la.VecDot(y, y)             // squared distance from origin in normalised space
			ova = b                          // ova ← y dot y
			oor = utl.GtePenalty(0, g(x), 1) // oor ← 0 ≥ g(x)

		// argmin_y{ β(y) + c(y(x)) | |g(x(y))| ≤ ϵ }
		case 4:
			y := ind.GetFloats()                      // must be inside ovfunc to avoid data race problems
			x := calc_orig_vars(ds, μ, σ, y)          // standard normal variables => original
			b := la.VecDot(y, y)                      // squared distance from origin in normalised space
			c := utl.GtePenalty(ϵ, math.Abs(g(x)), 1) // c ← ϵ ≥ |g(x)|
			ova = b + c                               // ova ← y dot y + c
			oor = c                                   // oor ← c

		default:
			chk.Panic("strategy %d is invalid", C.Strategy)
		}
		return
	}

	// transformation functions (for plotting)
	Tfcn := func(x []float64) (y []float64) { return calc_norm_vars(ds, μ, σ, x) }
	Tifcn := func(y []float64) (x []float64) { return calc_orig_vars(ds, μ, σ, y) }

	// evolver
	evo := goga.NewEvolverFloatChromo(C, vmin, vmax, ovfunc, goga.NewBingoFloats(vmin, vmax))

	// benchmarking
	cpu0 := time.Now()

	// for a number of trials
	betas := make([]float64, C.Ntrials)
	for i := 0; i < C.Ntrials; i++ {

		// reset population
		if i > 0 {
			for _, isl := range evo.Islands {
				isl.Pop.GenFloatRandom(C, vmin, vmax)
			}
		}
		pop0 := evo.Islands[0].Pop.GetCopy()

		// run
		check := i == C.Ntrials-1
		evo.Run(check, check)
		β := calc_beta(evo.Best, βref, ds, μ, σ, xref, C.Strategy, check)
		betas[i] = β

		// plot contour
		if check && C.DoPlot {
			pop1 := evo.Islands[0].Pop
			extra := func() { plt.SetXnticks(11); plt.SetYnticks(11) }
			istrans := C.Strategy > 2
			goga.PlotTwoVarsContour("/tmp/goga", io.Sf("rel-prob%d-ST%d-orig", C.Problem, C.Strategy), pop0, pop1, evo.Best, npts, extra, axequal,
				vmin, vmax, istrans, false, Tfcn, Tifcn, g, g)
			if len(ds) > 0 {
				goga.PlotTwoVarsContour("/tmp/goga", io.Sf("rel-prob%d-ST%d-tran", C.Problem, C.Strategy), pop0, pop1, evo.Best, npts, extra, axequal,
					vmin, vmax, istrans, true, Tfcn, Tifcn, g, g)
			}
		}
	}

	// benchmarking
	io.Pfcyan("\nelapsed time = %v\n", time.Now().Sub(cpu0))

	// analysis
	βmin, βave, βmax, βdev := rnd.StatBasic(betas, true)
	io.Pf("\nβmin = %v\n", βmin)
	io.PfYel("βave = %v\n", βave)
	io.Pf("βmax = %v\n", βmax)
	io.Pf("βdev = %v\n\n", βdev)
	io.Pf(rnd.BuildTextHist(nice_num(βmin-0.005), nice_num(βmax+0.005), 11, betas, "%.3f", 60))
}

// calc_orig_vars computes original variables from normal variables
func calc_orig_vars(ds []string, μ, σ, y []float64) (x []float64) {
	x = make([]float64, len(y))
	copy(x, y)
	if len(ds) == 0 { // all standard normal variables
		return
	}
	for i, typ := range ds {
		switch typ {
		case "nrm":
			x[i] = μ[i] + σ[i]*y[i]
		case "log": // TODO: check this
			σN := σ[i] * x[i]
			μN := (1 - math.Log(x[i]) + μ[i]) * x[i]
			x[i] = μN + σN*y[i]
		default:
			chk.Panic("distribution %q is not available", typ)
		}
	}
	return
}

// calc_norm_vars computes normal variables from original variables
func calc_norm_vars(ds []string, μ, σ, x []float64) (y []float64) {
	y = make([]float64, len(x))
	copy(y, x)
	if len(ds) == 0 { // all standard normal variables
		return
	}
	for i, typ := range ds {
		switch typ {
		case "nrm":
			y[i] = (x[i] - μ[i]) / σ[i]
		case "log": // TODO: check this
			σN := σ[i] * x[i]
			μN := (1 - math.Log(x[i]) + μ[i]) * x[i]
			y[i] = (x[i] - μN) / σN
		default:
			chk.Panic("distribution %q is not available", typ)
		}
	}
	return
}

// calc_beta calculates reliability index
func calc_beta(best *goga.Individual, βref float64, ds []string, μ, σ, xref []float64, strategy int, verbose bool) (β float64) {
	var xs, ys []float64
	if strategy < 3 {
		xs = best.GetFloats()             // check point
		ys = calc_norm_vars(ds, μ, σ, xs) // standard normal variables
	} else {
		ys = best.GetFloats()             // normal variables
		xs = calc_orig_vars(ds, μ, σ, ys) // check point
	}
	b := la.VecDot(ys, ys) // squared distance from origin in normalised space
	β = math.Sqrt(b)
	if verbose {
		io.Pf("\nova  = %g  oor = %g\n", best.Ova, best.Oor)
		io.Pf("x    = %v\n", xs)
		io.Pf("xref = %v\n", xref)
		io.PfYel("β    = %g", β)
		io.Pf(" (%g)\n", βref)
	}
	return
}

// nice_num returns a truncated float
func nice_num(x float64) float64 {
	s := io.Sf("%.2f", x)
	return io.Atof(s)
}
