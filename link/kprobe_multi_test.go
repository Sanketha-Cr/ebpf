//go:build !windows

package link

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/go-quicktest/qt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/internal/testutils"
	"github.com/cilium/ebpf/internal/unix"
)

var kprobeMultiSyms = []string{"vprintk", "inet6_release"}

func TestKprobeMulti(t *testing.T) {
	testutils.SkipIfNotSupported(t, haveBPFLinkKprobeMulti())

	prog := mustLoadProgram(t, ebpf.Kprobe, ebpf.AttachTraceKprobeMulti, "")

	km, err := KprobeMulti(prog, KprobeMultiOptions{Symbols: kprobeMultiSyms})
	if err != nil {
		t.Fatal(err)
	}
	defer km.Close()

	testLink(t, km, prog)
}

func TestKprobeMultiInput(t *testing.T) {
	// Program type that loads on all kernels. Not expected to link successfully.
	prog := mustLoadProgram(t, ebpf.SocketFilter, 0, "")

	// One of Symbols or Addresses must be given.
	_, err := KprobeMulti(prog, KprobeMultiOptions{})
	if !errors.Is(err, errInvalidInput) {
		t.Fatalf("expected errInvalidInput, got: %v", err)
	}

	// Symbols and Addresses are mutually exclusive.
	_, err = KprobeMulti(prog, KprobeMultiOptions{
		Symbols:   []string{"foo"},
		Addresses: []uintptr{1},
	})
	if !errors.Is(err, errInvalidInput) {
		t.Fatalf("expected errInvalidInput, got: %v", err)
	}

	// One Symbol, two cookies..
	_, err = KprobeMulti(prog, KprobeMultiOptions{
		Symbols: []string{"one"},
		Cookies: []uint64{2, 3},
	})
	if !errors.Is(err, errInvalidInput) {
		t.Fatalf("expected errInvalidInput, got: %v", err)
	}
}

func TestKprobeMultiErrors(t *testing.T) {
	testutils.SkipIfNotSupported(t, haveBPFLinkKprobeMulti())

	prog := mustLoadProgram(t, ebpf.Kprobe, ebpf.AttachTraceKprobeMulti, "")

	// Nonexistent kernel symbol.
	_, err := KprobeMulti(prog, KprobeMultiOptions{Symbols: []string{"bogus"}})
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, unix.EINVAL) {
		t.Fatalf("expected ErrNotExist or EINVAL, got: %s", err)
	}

	// Only have a negative test for addresses as it would be hard to maintain a
	// proper one.
	_, err = KprobeMulti(prog, KprobeMultiOptions{
		Addresses: []uintptr{^uintptr(0)},
	})
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, unix.EINVAL) {
		t.Fatalf("expected ErrNotExist or EINVAL, got: %s", err)
	}
}

func TestKprobeMultiCookie(t *testing.T) {
	testutils.SkipIfNotSupported(t, haveBPFLinkKprobeMulti())

	prog := mustLoadProgram(t, ebpf.Kprobe, ebpf.AttachTraceKprobeMulti, "")

	km, err := KprobeMulti(prog, KprobeMultiOptions{
		Symbols: kprobeMultiSyms,
		Cookies: []uint64{0, 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = km.Close()
}

func TestKprobeMultiProgramCall(t *testing.T) {
	if runtime.GOARCH == "s390x" {
		t.Skip("Skipping on s390x due to symbol resolution issues")
	}
	testutils.SkipIfNotSupported(t, haveBPFLinkKprobeMulti())

	m, p := newUpdaterMapProg(t, ebpf.Kprobe, ebpf.AttachTraceKprobeMulti)

	// For simplicity, just assert the increment happens with any symbol in the array.
	opts := KprobeMultiOptions{
		Symbols: []string{"__do_sys_getpid", "__do_sys_gettid"},
	}
	km, err := KprobeMulti(p, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Trigger ebpf program call.
	unix.Getpid()
	unix.Gettid()

	// Assert that the value got incremented to at least 2, while allowing
	// for bigger values, because we could race with other getpid/gettid
	// callers.
	assertMapValueGE(t, m, 0, 2)

	// Close the link.
	if err := km.Close(); err != nil {
		t.Fatal(err)
	}

	// Reset map value to 0 at index 0.
	if err := m.Update(uint32(0), uint32(0), ebpf.UpdateExist); err != nil {
		t.Fatal(err)
	}

	// Retrigger the ebpf program call.
	unix.Getpid()
	unix.Gettid()

	// Assert that this time the value has not been updated.
	assertMapValue(t, m, 0, 0)
}

func TestHaveBPFLinkKprobeMulti(t *testing.T) {
	testutils.CheckFeatureTest(t, haveBPFLinkKprobeMulti)
}

func TestKprobeSession(t *testing.T) {
	testutils.SkipIfNotSupported(t, haveBPFLinkKprobeMulti())

	prog := mustLoadProgram(t, ebpf.Kprobe, ebpf.AttachTraceKprobeSession, "")

	km, err := KprobeMulti(prog, KprobeMultiOptions{Symbols: kprobeMultiSyms, Session: true})
	testutils.SkipIfNotSupported(t, err)
	qt.Assert(t, qt.IsNil(err))
	defer km.Close()

	testLink(t, km, prog)
}

func TestHaveBPFLinkKprobeSession(t *testing.T) {
	testutils.CheckFeatureTest(t, haveBPFLinkKprobeSession)
}
