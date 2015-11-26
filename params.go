// Copyright 2015 Dorival de Moraes Pedroso. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goga

import (
	"encoding/json"

	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/io"
	"github.com/cpmech/gosl/rnd"
)

// Parameters hold all configuration parameters
type Parameters struct {

	// sizes
	Nova int // number of objective values
	Noor int // number of out-of-range values
	Nsol int // number of solutions in group
	Ngrp int // number of groups of solutions

	// time
	Tf    int // final time
	DtMig int // delta time for migration
	DtOut int // delta time for output

	// options
	Pll        bool    // parallel
	Seed       int     // seed for random numbers generator
	LatinDup   int     // Latin Hypercube duplicates number
	EpsMinProb float64 // minimum value for 'h' constraints
	Verbose    bool    // show messages
	Problem    int     // problem index

	// crossover and mutation
	DEpc    float64 // differential evolution pc
	DEmult  float64 // differential evolution multiplier
	DebEtac float64 // Deb's crossover parameter
	DebEtam float64 // Deb's mutation parameters
	PmFlt   float64 // probability of mutation for floats
	PmInt   float64 // probability of mutation for ints

	// range
	FltMin []float64 // minimum float allowed
	FltMax []float64 // maximum float allowed
	IntMin []int     // minimum int allowed
	IntMax []int     // maximum int allowed

	// derived
	NsolTot int       // total number of solutions = Nsol * Ngrp
	Nflt    int       // number of floats
	Nint    int       // number of integers
	DelFlt  []float64 // max float range
	DelInt  []int     // max int range
}

// Default sets default parameters
func (o *Parameters) Default() {

	// sizes
	o.Nova = 1
	o.Noor = 0
	o.Nsol = 24
	o.Ngrp = 4

	// time
	o.Tf = 100
	o.DtMig = o.Tf / 10
	o.DtOut = o.Tf / 5

	// options
	o.Pll = true
	o.Seed = 0
	o.LatinDup = 5
	o.EpsMinProb = 0.1
	o.Verbose = true
	o.Problem = 1

	// crossover and mutation
	o.DEpc = 0.1
	o.DEmult = 0.5
	o.DebEtac = 1
	o.DebEtam = 1
	o.PmFlt = 0.0
	o.PmInt = 0.1
}

// Read reads configuration parameters from JSON file
func (o *Parameters) Read(filenamepath string) {
	o.Default()
	b, err := io.ReadFile(filenamepath)
	if err != nil {
		chk.Panic("cannot read parameters file %q", filenamepath)
	}
	err = json.Unmarshal(b, o)
	if err != nil {
		chk.Panic("cannot unmarshal parameters file %q", filenamepath)
	}
	return
}

// CalcDerived computes derived variables and checks consistency
func (o *Parameters) CalcDerived() {

	// check
	if o.Nova < 1 {
		chk.Panic("number of objective values (nova) must be greater than 0")
	}
	if o.Ngrp < 1 {
		chk.Panic("at least one group must be defined. Ngrp=%d is incorrect", o.Ngrp)
	}
	if o.Nsol < 2 || (o.Nsol%2 != 0) {
		chk.Panic("number of solutions must be even and greater than 2. Nsol = %d is invalid", o.Nsol)
	}

	// derived
	o.NsolTot = o.Nsol * o.Ngrp
	if o.NsolTot%2 != 0 {
		chk.Panic("total number of solutions must be even. NsolTot = Nsol * Ngrp = %d is invalid", o.NsolTot)
	}
	o.Nflt = len(o.FltMin)
	o.Nint = len(o.IntMin)
	if o.Nflt == 0 && o.Nint == 0 {
		chk.Panic("either floats and ints must be set (via FltMin/Max or IntMin/Max)")
	}
	chk.IntAssert(len(o.FltMax), o.Nflt)
	chk.IntAssert(len(o.IntMax), o.Nint)
	o.DelFlt = make([]float64, o.Nflt)
	o.DelInt = make([]int, o.Nint)
	for i := 0; i < o.Nflt; i++ {
		o.DelFlt[i] = o.FltMax[i] - o.FltMin[i]
	}
	for i := 0; i < o.Nint; i++ {
		o.DelInt[i] = o.IntMax[i] - o.IntMin[i]
	}
	rnd.Init(o.Seed)
}

// EnforceRange makes sure x is within given range
func (o *Parameters) EnforceRange(i int, x float64) float64 {
	if x < o.FltMin[i] {
		return o.FltMin[i]
	}
	if x > o.FltMax[i] {
		return o.FltMax[i]
	}
	return x
}
