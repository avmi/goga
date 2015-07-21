// Copyright 2015 Dorival de Moraes Pedroso. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goga

import (
	"bytes"
	"sort"

	"github.com/cpmech/gosl/io"
	"github.com/cpmech/gosl/utl"
)

// Population holds all individuals
type Population []*Individual

// NewPopFloatChromo allocates a population made entirely of float point numbers
//  Input:
//   nbases -- number of bases in each float point gene
//   genes  -- all genes of all individuals [ninds][ngenes]
//  Output:
//   new population
func NewPopFloatChromo(nbases int, genes [][]float64) (pop Population) {
	ninds := len(genes)
	pop = make([]*Individual, ninds)
	for i := 0; i < ninds; i++ {
		pop[i] = new(Individual)
		pop[i].InitChromo(nbases, genes[i])
	}
	return
}

// NewPopReference creates a population based on a reference individual
//  Input:
//   ninds -- number of individuals to be generated
//   ref   -- reference individual with chromosome structure already set
//  Output:
//   new population
func NewPopReference(ninds int, ref *Individual) (pop Population) {
	pop = make([]*Individual, ninds)
	for i := 0; i < ninds; i++ {
		pop[i] = ref.GetCopy()
	}
	return
}

// NewPopRandom generates random population with individuals based on reference individual
// and gene values randomly drawn from Bingo.
//  Input:
//   ninds -- number of individuals to be generated
//   ref   -- reference individual with chromosome structure already set
//   bingo -- Bingo structure set with pool of values to draw gene values
//  Output:
//   new population
func NewPopRandom(ninds int, ref *Individual, bingo *Bingo) (pop Population) {
	pop = NewPopReference(ninds, ref)
	for i, ind := range pop {
		for j, g := range ind.Chromo {
			s := bingo.Draw(i, j, ninds)
			if g.Int != nil {
				g.SetInt(s.Int)
			}
			if g.Flt != nil {
				g.SetFloat(s.Flt)
			}
			if g.String != nil {
				g.SetString(s.String)
			}
			if g.Byte != nil {
				g.SetByte(s.Byte)
			}
			if g.Bytes != nil {
				g.SetBytes(s.Bytes)
			}
			if g.Func != nil {
				g.SetFunc(s.Func)
			}
		}
	}
	return
}

// Len returns the length of the population == number of individuals
func (o Population) Len() int {
	return len(o)
}

// Swap swaps two individuals
func (o Population) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

// Less returns true if 'i' is "less bad" than 'j'; therefore it can be used
// to sort the population in decreasing fitness order: best => worst
func (o Population) Less(i, j int) bool {
	return o[i].Fitness > o[j].Fitness
}

// Sort sorts the population from best to worst individuals; i.e. decreasing fitness values
func (o *Population) Sort() {
	sort.Sort(o)
}

// Output generates a nice table with population data
//  Input:
//  fmts -- [ngenes] formats for int, flt, string, byte, bytes, and func
//          use fmts == nil to choose default ones
func (o Population) Output(fmts [][]string) (buf *bytes.Buffer) {

	// check
	if len(o) < 1 {
		return
	}
	if len(o[0].Chromo) < 1 {
		return
	}

	// compute sizes and generate formats list
	ngenes := len(o[0].Chromo)
	sizes := utl.IntsAlloc(ngenes, 6)
	if fmts == nil {
		fmts = utl.StrsAlloc(ngenes, 6)
		for _, ind := range o {
			sz := ind.GetStringSizes()
			for i := 0; i < ngenes; i++ {
				for j := 0; j < 6; j++ {
					sizes[i][j] = imax(sizes[i][j], sz[i][j])
				}
			}
		}
		for i := 0; i < ngenes; i++ {
			for j, str := range []string{"d", "g", "s", "x", "s", "s"} {
				fmts[i][j] = io.Sf("%%%d%s", sizes[i][j]+1, str)
			}
		}
	}

	// compute sizes of header items
	nOvl, nFit := 0, 0
	for _, ind := range o {
		nOvl = imax(nOvl, len(io.Sf("%g", ind.ObjValue)))
		nFit = imax(nFit, len(io.Sf("%g", ind.Fitness)))
	}
	nOvl = imax(nOvl, 6) // 6 ==> len("ObjVal")
	nFit = imax(nFit, 7) // 7 ==> len("Fitness")

	// print individuals
	fmtOvl := io.Sf("%%%d", nOvl+1)
	fmtFit := io.Sf("%%%d", nFit+1)
	line, sza, szb := "", 0, 0
	for i, ind := range o {
		stra := io.Sf(fmtOvl+"g", ind.ObjValue) + io.Sf(fmtFit+"g", ind.Fitness) + " "
		strb := ind.Output(fmts)
		line += stra + strb + "\n"
		if i == 0 {
			sza, szb = len(stra), len(strb)
		}
	}

	// write to buffer
	fmtGen := io.Sf(" %%%d.%ds\n", szb, szb)
	n := sza + szb
	buf = new(bytes.Buffer)
	io.Ff(buf, printThickLine(n))
	io.Ff(buf, fmtOvl+"s", "ObjVal")
	io.Ff(buf, fmtFit+"s", "Fitness")
	io.Ff(buf, fmtGen, "Genes")
	io.Ff(buf, printThinLine(n))
	io.Ff(buf, line)
	io.Ff(buf, printThickLine(n))
	return
}

// OutFloatBases print bases of float genes
func (o Population) OutFloatBases(numFmt string) (l string) {
	for _, ind := range o {
		for _, g := range ind.Chromo {
			l += io.Sf(numFmt, g.Fbases)
		}
		l += "\n"
	}
	return
}
