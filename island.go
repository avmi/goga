// Copyright 2015 Dorival de Moraes Pedroso. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goga

import (
	"bytes"
	"math"

	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/graph"
	"github.com/cpmech/gosl/io"
	"github.com/cpmech/gosl/la"
	"github.com/cpmech/gosl/rnd"
	"github.com/cpmech/gosl/utl"
)

// constants
const (
	INF = 1e+30 // infinite distance
)

// Island holds one population and performs the reproduction operation
type Island struct {

	// input
	Id  int         // index of this island
	C   *ConfParams // configuration parameters
	Pop Population  // pointer to current population
	Bkp Population  // backup population

	// auxiliary internal data
	intmin  []int       // min int x-values
	intmax  []int       // max int x-values
	fltmin  []float64   // min flt x-values
	fltmax  []float64   // max flt x-values
	ovamin  []float64   // min ovas
	ovamax  []float64   // max ovas
	oormin  []float64   // min oors
	oormax  []float64   // max oors
	ovas    [][]float64 // all ova values
	oors    [][]float64 // all oor values
	sovas   [][]float64 // scaled ova values
	soors   [][]float64 // scaled oor values
	fitness []float64   // all fitness values
	prob    []float64   // probabilities
	cumprob []float64   // cumulated probabilities
	selinds []int       // indices of selected individuals
	A, B    []int       // indices of selected parents

	// crowding
	indices     []int           // [ninds] all indices of individuals
	groups      [][]int         // [ngroups][nparents] indices defining groups of individuals
	ndist       [][]float64     // neighgours distances [ninds][ninds]
	mdist       [][]float64     // [nparents][noffspring] matching distances
	match       graph.Munkres   // matches
	scores      ValIndDes       // scores
	competitors []*Individual   // [ngroups*nparents*nparents] all competitors
	cpparent    [][]*Individual // [ngroups][nparents] all parents (view to competitors)
	cpoffspr    [][]*Individual // [ngroups][noffspri] all offspring (view to competitors)

	// old crowding
	distR1    [][]float64   // [crowd_size][cowd_size] dist for round 1
	distR2    [][]float64   // [crowd_size][(crowd_size-1)*2] dist for round 2
	matchR1   graph.Munkres // matches for round 1
	matchR2   graph.Munkres // matches for round 2
	offspring []*Individual // offspring
	round2    []int         // ids for round 2

	// non-dominated front
	nfronts int     // number of non-dominated fronts
	fronts  [][]int // [ninds][ninds] non-dominated fronts (index of individuals)
	fsizes  []int   // [ninds] number of individuals in front
	idom    [][]int // [ninds][ninds] i dominate: index of individuals dominated by individual i
	sdom    []int   // [ninds] i dominate: size of individual i domination sublist
	ndby    []int   // [ninds] number of times individual i is dominated

	// results
	Report   bytes.Buffer // buffer to report results
	OutOvas  [][]float64  // [nova][ntimes] best objective values collected from multiple calls to SelectReprodAndRegen
	OutOors  [][]float64  // [noor][ntimes] best out-of-range values collected from multiple calls to SelectReprodAndRegen
	OutTimes []float64    // [ntimes] times corresponding to OutOvas and OutOors

	// statistics
	allbases [][]float64 // [ngenes*nbases][ninds] all bases
	devbases []float64   // [ngenes*nbases] deviations of bases
	larbases []float64   // [ngenes*nbases] largest bases; max(abs(bases))
	Nfeval   int         // number of objective function evaluations
}

// NewIsland creates a new island
func NewIsland(id int, C *ConfParams) (o *Island) {

	// check
	if C.Ninds < 2 || (C.Ninds%2 != 0) {
		chk.Panic("size of population must be even and greater than 2. C.Ninds = %d is invalid", C.Ninds)
	}
	if C.OvaOor == nil {
		chk.Panic("objective function (OvaOor) must be non nil")
	}

	// allocate island
	o = new(Island)
	o.Id = id
	o.C = C

	// create population
	if o.C.PopIntGen != nil {
		o.Pop = o.C.PopIntGen(id, o.C)
	}
	if o.C.PopFltGen != nil {
		o.Pop = o.C.PopFltGen(id, o.C)
	}
	if o.C.PopStrGen != nil {
		o.Pop = o.C.PopStrGen(id, o.C)
	}
	if o.C.PopKeyGen != nil {
		o.Pop = o.C.PopKeyGen(id, o.C)
	}
	if o.C.PopBytGen != nil {
		o.Pop = o.C.PopBytGen(id, o.C)
	}
	if o.C.PopFunGen != nil {
		o.Pop = o.C.PopFunGen(id, o.C)
	}
	if len(o.Pop) != o.C.Ninds {
		chk.Panic("generation of population failed:\nat least one generator function in Params must be non nil")
	}

	// copy population
	o.Bkp = o.Pop.GetCopy()

	// auxiliary data
	nints := len(o.Pop[0].Ints)
	nflts := o.Pop[0].Nfltgenes * o.Pop[0].Nbases
	if nints > 0 {
		o.intmin = make([]int, nints)
		o.intmax = make([]int, nints)
	}
	if nflts > 0 {
		o.fltmin = make([]float64, nflts)
		o.fltmax = make([]float64, nflts)
	}
	o.ovamin = make([]float64, o.C.Nova)
	o.ovamax = make([]float64, o.C.Nova)
	o.oormin = make([]float64, o.C.Noor)
	o.oormax = make([]float64, o.C.Noor)
	o.ovas = la.MatAlloc(o.C.Nova, o.C.Ninds)
	o.oors = la.MatAlloc(o.C.Noor, o.C.Ninds)
	o.sovas = la.MatAlloc(o.C.Nova, o.C.Ninds)
	o.soors = la.MatAlloc(o.C.Noor, o.C.Ninds)
	o.fitness = make([]float64, o.C.Ninds)
	o.prob = make([]float64, o.C.Ninds)
	o.cumprob = make([]float64, o.C.Ninds)
	o.selinds = make([]int, o.C.Ninds)
	o.A = make([]int, o.C.Ninds/2)
	o.B = make([]int, o.C.Ninds/2)

	// crowding
	if o.C.Ninds%o.C.NparGrp > 0 {
		chk.Panic("number of individuals must be multiple of NparGrp (number of parents in group)")
	}
	np := o.C.NparGrp    // number of parents in group
	no := np * (np - 1)  // number of offspring in group
	ng := o.C.Ninds / np // number of groups
	nr := np * np        // number of individuals in round (parents + offspring)
	nc := ng * nr        // total number of competitors
	nmax := utl.Imax(o.C.Ninds, nc)
	o.indices = utl.IntRange(o.C.Ninds)
	o.groups = utl.IntsAlloc(ng, np)
	o.ndist = la.MatAlloc(nmax, nmax)
	o.mdist = la.MatAlloc(np, no)
	o.match.Init(np, no)
	o.scores = make([]ValIndPair, nr)
	o.competitors = make([]*Individual, nc)
	for i := 0; i < nc; i++ {
		o.competitors[i] = o.Pop[0].GetCopy()
		o.competitors[i].Id = i
	}
	o.cpparent = make([][]*Individual, ng)
	o.cpoffspr = make([][]*Individual, ng)
	r := 0
	for k := 0; k < ng; k++ {
		o.cpparent[k] = make([]*Individual, np)
		o.cpoffspr[k] = make([]*Individual, no)
		s := 0
		for i := 0; i < np; i++ {
			o.cpparent[k][i], r = o.competitors[r], r+1
			for j := i + 1; j < np; j++ {
				o.cpoffspr[k][s], r, s = o.competitors[r], r+1, s+1
				o.cpoffspr[k][s], r, s = o.competitors[r], r+1, s+1
			}
		}
	}

	// old crowding
	n := o.C.NparGrp
	m := (o.C.NparGrp - 1) * 2
	if o.C.Ninds%n > 0 {
		chk.Panic("number of individuals must be multiple of crowd size")
	}
	o.distR1 = la.MatAlloc(n, m)
	o.matchR1.Init(n, m)
	if m-n > 0 {
		o.distR2 = la.MatAlloc(n, m-n)
		o.matchR2.Init(n, m-n)
		o.round2 = make([]int, m-n)
	}
	o.offspring = make([]*Individual, m)
	for i := 0; i < m; i++ {
		o.offspring[i] = o.Pop[0].GetCopy()
	}

	// non-dominated front
	o.fronts = make([][]int, nmax)
	o.fsizes = make([]int, nmax)
	o.idom = make([][]int, nmax)
	o.sdom = make([]int, nmax)
	o.ndby = make([]int, nmax)
	for i := 0; i < nmax; i++ {
		o.fronts[i] = make([]int, nmax)
		o.idom[i] = make([]int, nmax)
	}

	// compute objective values, demerits, and sort population
	o.CalcOvs(o.Pop, 0)
	o.CalcDemeritsCdistAndSort(o.Pop)

	// results
	o.OutOvas = la.MatAlloc(o.C.Nova, o.C.Tf)
	o.OutOors = la.MatAlloc(o.C.Noor, o.C.Tf)
	o.OutTimes = make([]float64, o.C.Tf)
	for i := 0; i < o.C.Nova; i++ {
		o.OutOvas[i][0] = o.Pop[0].Ovas[i]
	}
	for i := 0; i < o.C.Noor; i++ {
		o.OutOors[i][0] = o.Pop[0].Oors[i]
	}

	// stat
	if o.Pop[0].Nfltgenes > 0 {
		o.allbases = la.MatAlloc(nflts, o.C.Ninds)
		o.devbases = make([]float64, nflts)
		o.larbases = make([]float64, nflts)
	}
	return
}

// CalcOvs computes objective and out-of-range values
func (o *Island) CalcOvs(pop Population, time int) {
	for _, ind := range pop {
		o.C.OvaOor(ind, o.Id, time, &o.Report)
		o.Nfeval += 1
		for _, oor := range ind.Oors {
			if oor < 0 {
				chk.Panic("out-of-range values must be positive (or zero) indicating the positive distance to constraints. oor=%g is invalid", oor)
			}
		}
	}
}

// CalcDemeritsCdistAndSort computes demerits and sort population
func (o *Island) CalcDemeritsCdistAndSort(pop Population) {

	// fill auxiliary arrays
	for i, ind := range pop {

		// set auxiliary arrays
		for j := 0; j < o.C.Nova; j++ {
			o.ovas[j][i] = ind.Ovas[j]
		}
		for j := 0; j < o.C.Noor; j++ {
			o.oors[j][i] = ind.Oors[j]
		}

		// calc int limits
		for j, x := range ind.Ints {
			if i == 0 {
				o.intmin[j], o.intmax[j] = x, x
			} else {
				o.intmin[j] = utl.Imin(o.intmin[j], x)
				o.intmax[j] = utl.Imax(o.intmax[j], x)
			}
		}

		// calc flt limits
		for j, x := range ind.Floats {
			if i == 0 {
				o.fltmin[j], o.fltmax[j] = x, x
			} else {
				o.fltmin[j] = utl.Min(o.fltmin[j], x)
				o.fltmax[j] = utl.Max(o.fltmax[j], x)
			}
		}
	}

	// compute scaled values and min/max ovas/oors
	for i := 0; i < o.C.Nova; i++ {
		o.ovamin[i], o.ovamax[i] = utl.Scaling(o.sovas[i], o.ovas[i], 0, 1e-16, false, true)
	}
	for i := 0; i < o.C.Noor; i++ {
		o.oormin[i], o.oormax[i] = utl.Scaling(o.soors[i], o.oors[i], 0, 1e-16, false, true)
	}

	// sort and compute distances
	o.NomDomSortAndCalcDistances(pop)

	// compute demerit values
	for i, ind := range pop {
		ind.Demerit = 0
		if o.C.Nova > 1 {
			ind.Demerit = float64(ind.FrontId)
		} else {
			for j := 0; j < o.C.Nova; j++ {
				ind.Demerit += o.sovas[j][i]
			}
		}
	}
	shift := 2.0
	for i, ind := range pop {
		firstOor := true
		for j := 0; j < o.C.Noor; j++ {
			if ind.Oors[j] > 0 {
				if firstOor {
					ind.Demerit = shift
					firstOor = false
				}
				ind.Demerit += o.soors[j][i]
			}
		}
	}

	// sort population with respect to demerit values
	pop.SortByDemerit()
}

// Run runs evolutionary process
func (o *Island) Run(time int, doreport, verbose bool) {

	// run
	switch o.C.GAtype {
	case "crowd":
		o.update_crowding(time)
	case "cold":
		o.update_crowding_old(time)
	default:
		o.update_standard(time)
	}

	// swap populations (Pop will always point to current one)
	o.Pop, o.Bkp = o.Bkp, o.Pop
	o.CalcDemeritsCdistAndSort(o.Pop)

	// elitism
	if o.C.Elite {
		prev_best, cur_worst := o.Bkp[0], o.Pop[o.C.Ninds-1]
		prev_dominates, _ := IndCompareDet(prev_best, cur_worst)
		if prev_dominates {
			prev_best.CopyInto(cur_worst)
		}
	}

	// statistics and regeneration of float-point individuals
	var averho float64
	if o.Pop[0].Nfltgenes > 0 {
		_, averho, _, _ = o.FltStat()
		homogeneous := averho < o.C.RegTol
		if homogeneous {
			o.Regenerate(time)
			if doreport {
				io.Ff(&o.Report, "time=%d: regeneration\n", time)
			}
			if verbose {
				io.Pfmag(" .")
			}
		}
	}

	// report
	if doreport {
		o.WritePopToReport(time, averho)
	}

	// post-process
	if o.C.PostProc != nil {
		o.C.PostProc(o.Id, time, o.Pop)
	}

	// results
	for i := 0; i < o.C.Nova; i++ {
		o.OutOvas[i][time] = o.Pop[0].Ovas[i]
	}
	for i := 0; i < o.C.Noor; i++ {
		o.OutOors[i][time] = o.Pop[0].Oors[i]
	}
	o.OutTimes[time] = float64(time)
}

// update_crowding runs the evolutionary process with niching via crowding and tournament selection
func (o *Island) update_crowding(time int) {

	// select groups
	rnd.IntGetGroups(o.groups, o.indices)

	// auxiliary variables
	np := o.C.NparGrp    // number of parents in group
	no := np * (np - 1)  // number of offspring in group
	ng := o.C.Ninds / np // number of groups
	nr := np * np        // number of individuals in round (parents + offspring)
	//nc := ng * nr        // total number of competitors

	// set parents
	for k := 0; k < ng; k++ {
		for i := 0; i < np; i++ {
			o.Pop[o.groups[k][i]].CopyInto(o.cpparent[k][i])
		}
	}

	// create offspring and set competitors
	var a, b, A, B, C, D *Individual
	for k := 0; k < ng; k++ {
		s := 0
		for i := 0; i < np; i++ {
			A = o.cpparent[k][i]
			for j := i + 1; j < np; j++ {
				B = o.cpparent[k][j]
				if o.C.Ops.Use4inds {
					knext := (k + 1) % ng
					if false {
						next := rnd.IntGetUniqueN(0, np, 2)
						C = o.Pop[o.groups[knext][next[0]]]
						D = o.Pop[o.groups[knext][next[1]]]
					} else {
						C = o.Pop[o.groups[knext][0]]
						D = o.Pop[o.groups[knext][1]]
					}
					//o.four_nondom(A, B, C, D)
				}
				a, s = o.cpoffspr[k][s], s+1
				b, s = o.cpoffspr[k][s], s+1
				IndCrossover(a, b, A, B, C, D, time, &o.C.Ops)
				IndMutation(a, time, &o.C.Ops)
				IndMutation(b, time, &o.C.Ops)
				o.C.OvaOor(a, o.Id, time+1, &o.Report)
				o.C.OvaOor(b, o.Id, time+1, &o.Report)
				o.Nfeval += 2
			}
		}
	}

	// calc non-dominated front, sort and compute distances
	o.NomDomSortAndCalcDistances(o.competitors)

	// tournaments: all versus all
	idxnew := 0
	if o.C.AllVsAll {
		for k := 0; k < ng; k++ {

			// reset scores
			for i := 0; i < nr; i++ {
				o.scores[i].Val = 0
			}

			// matches
			r := k * nr
			round := o.competitors[r : r+nr]
			for i := 0; i < nr; i++ {
				A = round[i]
				o.scores[i].Ind = A
				for j := i + 1; j < nr; j++ {
					B = round[j]
					o.scores[j].Ind = B
					if o.tournament(A, B) {
						o.scores[i].Val += 1 // A wins
					} else {
						o.scores[j].Val += 1 // B wins
					}
				}
			}

			// winners
			o.scores.Sort()
			for i := 0; i < np; i++ {
				o.scores[i].Ind.CopyInto(o.Bkp[idxnew])
				idxnew++
			}
		}
	}

	// tournaments: using match distance
	if !o.C.AllVsAll {
		for k := 0; k < ng; k++ {

			// compute match distances
			for i := 0; i < np; i++ {
				A = o.cpparent[k][i]
				for j := 0; j < no; j++ {
					a = o.cpoffspr[k][j]
					o.mdist[i][j] = IndDistance(A, a, nil, nil, nil, nil, o.ovamin, o.ovamax, true)
				}
			}
			//la.PrintMat("mdist", o.mdist, "%8.5f", false)

			// match competitors
			o.match.SetCostMatrix(o.mdist)
			o.match.Run()

			// matches
			for i := 0; i < np; i++ {
				A = o.cpparent[k][i]
				B = o.cpoffspr[k][o.match.Links[i]]
				if o.tournament(A, B) {
					A.CopyInto(o.Bkp[idxnew]) // A wins
				} else {
					B.CopyInto(o.Bkp[idxnew]) // B wins
				}
				idxnew++
			}
		}
	}
}

func (o *Island) update_crowding_old(time int) {

	// select groups
	rnd.IntGetGroups(o.groups, o.indices)

	// auxiliary variables
	n := o.C.NparGrp
	m := (o.C.NparGrp - 1) * 2
	ncrowd := len(o.groups)

	// run tournaments
	for igroup, group := range o.groups {

		// crossover, mutation and new objective values
		for r := 0; r < n-1; r++ {
			i, j := r, r+1
			k, l := r*2, r*2+1
			I, J := group[i], group[j]
			A, B := o.Pop[I], o.Pop[J]
			a, b := o.offspring[k], o.offspring[l]
			if o.C.Ops.Use4inds {
				jgroup := (igroup + 1) % ncrowd
				C, D := o.Pop[o.groups[jgroup][0]], o.Pop[o.groups[jgroup][1]]
				o.four_nondom(A, B, C, D)
				IndCrossover(a, b, A, B, C, D, time, &o.C.Ops)
			} else {
				IndCrossover(a, b, A, B, nil, nil, time, &o.C.Ops)
			}
			IndMutation(a, time, &o.C.Ops)
			IndMutation(b, time, &o.C.Ops)
			o.C.OvaOor(a, o.Id, time+1, &o.Report)
			o.C.OvaOor(b, o.Id, time+1, &o.Report)
			o.Nfeval += 2
		}

		// round 1: compute distances
		for i := 0; i < n; i++ {
			I := group[i]
			A := o.Pop[I]
			for j := 0; j < m; j++ {
				B := o.offspring[j]
				o.distR1[i][j] = IndDistance(A, B, o.intmin, o.intmax, o.fltmin, o.fltmax, o.ovamin, o.ovamax, o.C.DistOvs)
			}
		}

		// round 1: match competitors
		o.matchR1.SetCostMatrix(o.distR1)
		o.matchR1.Run()

		// compute next round
		k := 0
		for i := 0; i < m; i++ {
			if utl.IntIndexSmall(o.matchR1.Links, i) < 0 {
				o.round2[k] = i
				k++
			}
		}

		// round 1: tournament
		for i := 0; i < n; i++ {
			I := group[i]
			j := o.matchR1.Links[i]
			A, B := o.Pop[I], o.offspring[j]
			o.tournament_old(A, B, I)
		}

		// next round
		if m-n > 0 {

			// round 2: compute distances
			for i := 0; i < n; i++ {
				I := group[i]
				A := o.Bkp[I]
				for j := 0; j < m-n; j++ {
					J := o.round2[j]
					B := o.offspring[J]
					o.distR2[i][j] = IndDistance(A, B, o.intmin, o.intmax, o.fltmin, o.fltmax, o.ovamin, o.ovamax, o.C.DistOvs)
				}
			}

			// round 2: match competitors
			o.matchR2.SetCostMatrix(o.distR2)
			o.matchR2.Run()

			// round 2: tournament
			for i := 0; i < n; i++ {
				I := group[i]
				k := o.matchR2.Links[i]
				if k >= 0 {
					j := o.round2[k]
					A, B := o.Bkp[I], o.offspring[j]
					o.tournament_old(A, B, I)
				}
			}
		}
	}
}

// tournament performs tournament; B_wins = !A_wins
//  Note: crowd dist (Cdist) must be set already
func (o *Island) tournament(A, B *Individual) (A_wins bool) {
	if o.C.CompProb {
		if IndCompareProb(A, B, o.C.ParetoPhi) {
			return true
		}
		return false
	}
	A_dom, B_dom := IndCompareDet(A, B)
	if A_dom {
		return true
	}
	if B_dom {
		return false
	}
	if (A.FrontId != B.FrontId) || o.C.CdistOff {
		prob := 0.5
		if A.DistNeigh > B.DistNeigh {
			prob = 0.6
			//io.Pforan("A(win):ndist=%25g B     :ndist=%25g\n", A.Ndist, B.Ndist)
			//return true
		}
		if B.DistNeigh > A.DistNeigh {
			//io.Pfyel("A     :ndist=%25g B(win):ndist=%25g\n", A.Ndist, B.Ndist)
			//return false
			prob = 0.4
		}
		if rnd.FlipCoin(prob) {
			return true
		}
		return false
	}
	if A.DistCrowd > B.DistCrowd {
		//io.Pforan("A.Front=%d B.Frond=%d A.Cdist=%8.3f B.Cdist=%8.3f\n", A.FrontId, B.FrontId, A.Cdist, B.Cdist)
		return true
	}
	//if A.Cdist > 1e+29 || B.Cdist > 1e+29 {
	//io.Pfblue2("%v %v => Cdist=%v %v\n", A.FrontId, B.FrontId, A.Cdist, B.Cdist)
	//chk.Panic("stop")
	//}
	if A.DistCrowd == B.DistCrowd {
		//io.Pforan("A.Cdist=%v B.Cdist=%v\n", A.Cdist, B.Cdist)
		prob := 0.5
		if A.DistNeigh > B.DistNeigh {
			prob = 0.6
			//io.Pforan("A(win):ndist=%25g B     :ndist=%25g\n", A.Ndist, B.Ndist)
			//return true
		}
		if B.DistNeigh > A.DistNeigh {
			//io.Pfyel("A     :ndist=%25g B(win):ndist=%25g\n", A.Ndist, B.Ndist)
			//return false
			prob = 0.4
		}
		if rnd.FlipCoin(prob) {
			return true
		}
	}
	return false
}

// tournament runs game between A and B
func (o *Island) tournament_old(A, B *Individual, saveInto int) {

	// probabilistic
	if o.C.CompProb {
		if IndCompareProb(A, B, o.C.ParetoPhi) {
			A.CopyInto(o.Bkp[saveInto]) // A wins
			return
		}
		B.CopyInto(o.Bkp[saveInto]) // B wins
		return
	}

	// deterministic
	A_dom, B_dom := IndCompareDet(A, B)
	if A_dom {
		A.CopyInto(o.Bkp[saveInto]) // A wins
		return
	}
	if B_dom {
		B.CopyInto(o.Bkp[saveInto]) // B wins
		return
	}
	if rnd.FlipCoin(0.5) { // tie => roll dice
		A.CopyInto(o.Bkp[saveInto]) // A wins by chance
		return
	}
	B.CopyInto(o.Bkp[saveInto]) // B wins by chance
}

// update_standard performs the selection, reproduction and regeneration processes
//  Note: this function considers a SORTED population already
func (o *Island) update_standard(time int) {

	// fitness
	ninds := len(o.Pop)
	var sumfit float64
	if o.C.Rnk { // ranking
		sp := o.C.RnkSp
		for i := 0; i < ninds; i++ {
			o.fitness[i] = 2.0 - sp + 2.0*(sp-1.0)*float64(ninds-i-1)/float64(ninds-1)
			sumfit += o.fitness[i]
		}
	} else {
		mindem := o.Pop[0].Demerit
		maxdem := mindem
		for i := 0; i < ninds; i++ {
			mindem = utl.Min(mindem, o.Pop[i].Demerit)
			maxdem = utl.Max(maxdem, o.Pop[i].Demerit)
		}
		for i, ind := range o.Pop {
			o.fitness[i] = (maxdem - ind.Demerit) / (maxdem - mindem)
			sumfit += o.fitness[i]
		}
	}

	// probabilities
	for i := 0; i < ninds; i++ {
		o.prob[i] = o.fitness[i] / sumfit
		if i == 0 {
			o.cumprob[i] = o.prob[i]
		} else {
			o.cumprob[i] = o.cumprob[i-1] + o.prob[i]
		}
	}

	// selection
	if o.C.Rws {
		RouletteSelect(o.selinds, o.cumprob, nil)
	} else {
		SUSselect(o.selinds, o.cumprob, -1)
	}
	FilterPairs(o.A, o.B, o.selinds)

	// reproduction
	h := ninds / 2
	for i := 0; i < ninds/2; i++ {
		IndCrossover(o.Bkp[i], o.Bkp[h+i], o.Pop[o.A[i]], o.Pop[o.B[i]], nil, nil, time, &o.C.Ops)
		IndMutation(o.Bkp[i], time, &o.C.Ops)
		IndMutation(o.Bkp[h+i], time, &o.C.Ops)
	}

	// compute objective values
	o.CalcOvs(o.Bkp, time+1) // +1 => this is an updated generation
}

// auxiliary //////////////////////////////////////////////////////////////////////////////////////

// Regenerate regenerates population with basis on best individual(s)
func (o *Island) Regenerate(time int) {
	ninds := len(o.Pop)
	start := ninds - int(o.C.RegPct*float64(ninds))
	for i := start; i < ninds; i++ {
		for j := 0; j < o.Pop[i].Nfltgenes; j++ {
			xmin, xmax := o.C.RangeFlt[j][0], o.C.RangeFlt[j][1]
			o.Pop[i].SetFloat(j, rnd.Float64(xmin, xmax))
		}
	}
	o.CalcOvs(o.Pop, time)
	o.CalcDemeritsCdistAndSort(o.Pop)
	return
}

// FltStat computes some statistic information with float-point individuals
//  rho (ρ) is a normalised quantity measuring the deviation of bases of each gene
func (o *Island) FltStat() (minrho, averho, maxrho, devrho float64) {
	ngenes, nbases := o.Pop[0].Nfltgenes, o.Pop[0].Nbases
	for k, ind := range o.Pop {
		for i := 0; i < ngenes; i++ {
			for j := 0; j < nbases; j++ {
				x := ind.Floats[i*nbases+j]
				o.allbases[i*nbases+j][k] = x
				if k == 0 {
					o.larbases[i*nbases+j] = math.Abs(x)
				} else {
					o.larbases[i*nbases+j] = utl.Max(o.larbases[i*nbases+j], math.Abs(x))
				}
			}
		}
	}
	for i := 0; i < ngenes; i++ {
		for j := 0; j < nbases; j++ {
			normfactor := 1.0 + o.larbases[i*nbases+j]
			o.devbases[i*nbases+j] = rnd.StatDev(o.allbases[i*nbases+j], o.C.UseStdDev) / normfactor
		}
	}
	minrho, averho, maxrho, devrho = rnd.StatBasic(o.devbases, o.C.UseStdDev)
	return
}

// WritePopToReport writes population to report
func (o *Island) WritePopToReport(time int, averho float64) {
	io.Ff(&o.Report, "time=%d averho=%g\n", time, averho)
	o.Report.Write(o.Pop.Output(o.C).Bytes())
}

// SaveReport saves report to file
func (o Island) SaveReport(verbose bool) {
	dosave := o.C.FnKey != ""
	if dosave {
		if o.C.DirOut == "" {
			o.C.DirOut = "/tmp/goga"
		}
		if verbose {
			io.WriteFileVD(o.C.DirOut, io.Sf("%s-%d.rpt", o.C.FnKey, o.Id), &o.Report)
			return
		}
		io.WriteFileD(o.C.DirOut, io.Sf("%s-%d.rpt", o.C.FnKey, o.Id), &o.Report)
	}
}

// four_nondom finds 2 nondominated individuals among 4 individuals
func (o *Island) four_nondom(A, B, C, D *Individual) {
	var Bdom, Cdom, Ddom bool
	Bdom, _ = IndCompareDet(B, A)
	if Bdom {
		B, A = A, B
	}
	Cdom, Bdom = IndCompareDet(C, B)
	if Cdom {
		C, B = B, C
	}
	Ddom, Cdom = IndCompareDet(D, C)
	if Ddom {
		D, C = C, D
	}
	Bdom, _ = IndCompareDet(B, A)
	if Bdom {
		B, A = A, B
	}
	Cdom, Bdom = IndCompareDet(C, B)
	if Cdom {
		C, B = B, C
	}
	Bdom, _ = IndCompareDet(B, A)
	if Bdom {
		B, A = A, B
	}
}
