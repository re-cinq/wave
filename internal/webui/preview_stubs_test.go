//go:build webui_preview

package webui

import (
	"reflect"
	"testing"
)

func TestPreviewLandingStubDeterministic(t *testing.T) {
	s := previewLandingStub{}
	a, err := s.Landing()
	if err != nil {
		t.Fatalf("first call err = %v", err)
	}
	b, err := s.Landing()
	if err != nil {
		t.Fatalf("second call err = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("landing stub non-deterministic: %#v vs %#v", a, b)
	}
}

func TestPreviewOnboardStubDeterministic(t *testing.T) {
	s := previewOnboardStub{}
	a, err := s.Onboard()
	if err != nil {
		t.Fatalf("first call err = %v", err)
	}
	b, err := s.Onboard()
	if err != nil {
		t.Fatalf("second call err = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("onboard stub non-deterministic: %#v vs %#v", a, b)
	}
}

func TestPreviewWorkStubDeterministic(t *testing.T) {
	s := previewWorkStub{}
	a, err := s.Work()
	if err != nil {
		t.Fatalf("first call err = %v", err)
	}
	b, err := s.Work()
	if err != nil {
		t.Fatalf("second call err = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("work stub non-deterministic: %#v vs %#v", a, b)
	}
}

func TestPreviewWorkItemStubDeterministic(t *testing.T) {
	s := previewWorkItemStub{}
	a, err := s.WorkItem()
	if err != nil {
		t.Fatalf("first call err = %v", err)
	}
	b, err := s.WorkItem()
	if err != nil {
		t.Fatalf("second call err = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("work-item stub non-deterministic: %#v vs %#v", a, b)
	}
}

func TestPreviewProposalStubDeterministic(t *testing.T) {
	s := previewProposalStub{}
	a, err := s.Proposal()
	if err != nil {
		t.Fatalf("first call err = %v", err)
	}
	b, err := s.Proposal()
	if err != nil {
		t.Fatalf("second call err = %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("proposal stub non-deterministic: %#v vs %#v", a, b)
	}
}

func TestPreviewStubsReturnNoError(t *testing.T) {
	cases := []struct {
		name string
		call func() error
	}{
		{"landing", func() error { _, err := previewLandingStub{}.Landing(); return err }},
		{"onboard", func() error { _, err := previewOnboardStub{}.Onboard(); return err }},
		{"work", func() error { _, err := previewWorkStub{}.Work(); return err }},
		{"work_item", func() error { _, err := previewWorkItemStub{}.WorkItem(); return err }},
		{"proposal", func() error { _, err := previewProposalStub{}.Proposal(); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
		})
	}
}

func TestDefaultPreviewServicesAllPopulated(t *testing.T) {
	r := defaultPreviewServices()
	if r == nil {
		t.Fatal("registry nil")
	}
	if r.Landing == nil {
		t.Error("Landing nil")
	}
	if r.Onboard == nil {
		t.Error("Onboard nil")
	}
	if r.Work == nil {
		t.Error("Work nil")
	}
	if r.WorkItem == nil {
		t.Error("WorkItem nil")
	}
	if r.Proposal == nil {
		t.Error("Proposal nil")
	}
}
