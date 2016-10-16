package rnntest

import (
	"math"
	"testing"

	"github.com/unixpickle/autofunc"
	"github.com/unixpickle/autofunc/functest"
	"github.com/unixpickle/num-analysis/linalg"
	"github.com/unixpickle/weakai/rnn"
)

// A BlockChecker performs gradient checking and other
// edge-case checks on a Block.
type BlockChecker struct {
	// B is the Block to test.
	B rnn.Block

	// Input is the list of input sequences to use.
	Input [][]*autofunc.Variable

	// Vars is the list of variables whose gradients should
	// be checked.
	Vars []*autofunc.Variable

	// RV stores the first derivatives of any relevant
	// variables from Vars.
	RV autofunc.RVector

	// Delta is the delta used for gradient approximation.
	// If it is 0, functest.DefaultDelta is used.
	Delta float64

	// Prec is the precision to use when comparing values.
	// If it is 0, functest.DefaultPrec is used.
	Prec float64
}

// FullCheck performs gradient checking and checks  for
// some edge cases.
func (b *BlockChecker) FullCheck(t *testing.T) {
	seqChecker := &functest.SeqRFuncChecker{
		F:     &rnn.BlockSeqFunc{B: b.B},
		Input: b.Input,
		Vars:  b.Vars,
		RV:    b.RV,
		Delta: b.Delta,
		Prec:  b.Prec,
	}
	seqChecker.FullCheck(t)
	b.testNilUpstream(t)
	b.testNilUpstreamR(t)
}

func (b *BlockChecker) testNilUpstream(t *testing.T) {
	t.Run("Nil Upstream", func(t *testing.T) {
		out := b.B.ApplyBlock([]rnn.State{b.B.StartState()},
			[]autofunc.Result{b.Input[0][0]})
		g1 := autofunc.NewGradient(b.Vars)
		initLen1 := len(g1)
		out.PropagateGradient(nil, nil, g1)

		g2 := autofunc.NewGradient(b.Vars)
		initLen2 := len(g2)
		zeroUpstream := make([]linalg.Vector, len(out.Outputs()))
		for i, x := range out.Outputs() {
			zeroUpstream[i] = make(linalg.Vector, len(x))
		}
		nilStateUpstream := make([]rnn.StateGrad, len(out.States()))
		out.PropagateGradient(zeroUpstream, nilStateUpstream, g2)

		if len(g1) != initLen1 {
			t.Errorf("all nil gradient length changed from %d to %d", initLen1, len(g1))
		}
		if len(g2) != initLen2 {
			t.Errorf("non-nil gradient length changed from %d to %d", initLen2, len(g2))
		}

		for i, variable := range b.Vars {
			val1 := g1[variable]
			val2 := g2[variable]
			if !b.vecsEqual(val1, val2) {
				t.Errorf("gradients for var %d don't match: %v and %v", i, val1, val2)
			}
		}
	})
}

func (b *BlockChecker) testNilUpstreamR(t *testing.T) {
	t.Run("Nil Upstream R", func(t *testing.T) {
		out := b.B.ApplyBlockR(b.RV, []rnn.RState{b.B.StartRState(b.RV)},
			[]autofunc.RResult{autofunc.NewRVariable(b.Input[0][0], b.RV)})
		g1 := autofunc.NewGradient(b.Vars)
		rg1 := autofunc.NewRGradient(b.Vars)
		initLen1 := len(g1)
		out.PropagateRGradient(nil, nil, nil, rg1, g1)
		g2 := autofunc.NewGradient(b.Vars)
		rg2 := autofunc.NewRGradient(b.Vars)
		initLen2 := len(g2)

		zeroUpstream := make([]linalg.Vector, len(out.Outputs()))
		for i, x := range out.Outputs() {
			zeroUpstream[i] = make(linalg.Vector, len(x))
		}
		nilStateUpstream := make([]rnn.RStateGrad, len(out.RStates()))
		out.PropagateRGradient(zeroUpstream, zeroUpstream, nilStateUpstream, rg2, g2)

		if len(g1) != initLen1 {
			t.Errorf("all nil gradient length changed from %d to %d", initLen1, len(g1))
		}
		if len(rg1) != initLen1 {
			t.Errorf("all nil r-gradient length changed from %d to %d", initLen1, len(rg1))
		}
		if len(g2) != initLen2 {
			t.Errorf("non-nil gradient length changed from %d to %d", initLen2, len(g2))
		}
		if len(rg2) != initLen2 {
			t.Errorf("non-nil r-gradient length changed from %d to %d", initLen2, len(rg2))
		}

		for i, variable := range b.Vars {
			val1 := g1[variable]
			val2 := g2[variable]
			if !b.vecsEqual(val1, val2) {
				t.Errorf("gradients for var %d don't match: %v and %v", i, val1, val2)
			}
			val1 = rg1[variable]
			val2 = rg2[variable]
			if !b.vecsEqual(val1, val2) {
				t.Errorf("r-gradients for var %d don't match: %v and %v", i, val1, val2)
			}
		}
	})
}

func (b *BlockChecker) vecsEqual(v1, v2 linalg.Vector) bool {
	if len(v1) != len(v2) {
		return false
	}
	prec := b.Prec
	if prec == 0 {
		prec = functest.DefaultPrec
	}
	for i, x := range v1 {
		y := v2[i]
		if math.IsNaN(x) != math.IsNaN(y) || math.Abs(x-y) > prec {
			return false
		}
	}
	return true
}
