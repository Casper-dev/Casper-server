package main

import (
	"testing"

	"gitlab.com/casperDev/Casper-server/commands"
)

func TestIsCientErr(t *testing.T) {
	t.Log("Catch both pointers and values")
	if !isClientError(commands.Error{Code: commands.ErrClient}) {
		t.Errorf("misidentified value")
	}
	if !isClientError(&commands.Error{Code: commands.ErrClient}) {
		t.Errorf("misidentified pointer")
	}
}
