package testscenario

// Features implemented: testing-framework/test-runner

import "testing"

func TestExecContext_storeAndResolveContext(t *testing.T) {
	ctx := NewExecContext()
	if err := ctx.StoreOutput("create", "project_id", "p-123", StoreContext); err != nil {
		t.Fatal(err)
	}
	val, err := ctx.ResolveVar("context.project_id")
	if err != nil {
		t.Fatal(err)
	}
	if val != "p-123" {
		t.Errorf("got %q, want %q", val, "p-123")
	}
}

func TestExecContext_storeAndResolveStep(t *testing.T) {
	ctx := NewExecContext()
	if err := ctx.StoreOutput("create", "raw", "data", StoreStep); err != nil {
		t.Fatal(err)
	}
	val, err := ctx.ResolveVar("steps.create.outputs.raw")
	if err != nil {
		t.Fatal(err)
	}
	if val != "data" {
		t.Errorf("got %q, want %q", val, "data")
	}
}

func TestExecContext_storeBoth(t *testing.T) {
	ctx := NewExecContext()
	if err := ctx.StoreOutput("s1", "id", "x", StoreBoth); err != nil {
		t.Fatal(err)
	}
	v1, _ := ctx.ResolveVar("context.id")
	v2, _ := ctx.ResolveVar("steps.s1.outputs.id")
	if v1 != "x" || v2 != "x" {
		t.Errorf("context=%q step=%q, both should be %q", v1, v2, "x")
	}
}

func TestExecContext_contextKeyOverwrite(t *testing.T) {
	ctx := NewExecContext()
	_ = ctx.StoreOutput("s1", "id", "x", StoreContext)
	if err := ctx.StoreOutput("s2", "id", "y", StoreContext); err != nil {
		t.Fatalf("overwriting context key should succeed: %v", err)
	}
	val, _ := ctx.ResolveVar("context.id")
	if val != "y" {
		t.Errorf("got %q, want %q (latest value)", val, "y")
	}
}

func TestExecContext_resolveInString(t *testing.T) {
	ctx := NewExecContext()
	_ = ctx.StoreOutput("create", "pid", "p-1", StoreContext)
	result, err := ctx.ResolveString("synchestra remove --id ${{ context.pid }}")
	if err != nil {
		t.Fatal(err)
	}
	if result != "synchestra remove --id p-1" {
		t.Errorf("got %q", result)
	}
}

func TestExecContext_resolveUnknownVar(t *testing.T) {
	ctx := NewExecContext()
	_, err := ctx.ResolveVar("context.missing")
	if err == nil {
		t.Fatal("expected error for unknown variable")
	}
}
