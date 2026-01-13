package main

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
)

type mockDockerClient struct {
	containers  []container.Summary
	removeErr   map[string]error
	removedIDs  []string
	listErr     error
	closeCalled bool
}

func (m *mockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return m.containers, m.listErr
}

func (m *mockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	m.removedIDs = append(m.removedIDs, containerID)
	if m.removeErr == nil {
		return nil
	}

	if err, ok := m.removeErr[containerID]; ok {
		return err
	}

	return nil
}

func (m *mockDockerClient) Close() error {
	m.closeCalled = true
	return nil
}

func TestCleanAutoNamedContainersRemovesMatches(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClient{
		containers: []container.Summary{
			{ID: "id1", Names: []string{"/inspiring_franklin"}},
			{ID: "id2", Names: []string{"/custom_app"}},
			{ID: "id3", Names: []string{"/agitated_wescoff7"}},
		},
		removeErr: map[string]error{
			"id3": errdefs.NotFound(errors.New("already gone")),
		},
	}

	removed, err := CleanAutoNamedContainers(ctx, mock)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(removed) != 1 || removed[0] != "inspiring_franklin" {
		t.Fatalf("unexpected removed list: %v", removed)
	}

	if containsID(mock.removedIDs, "id2") {
		t.Fatalf("custom container should not be removed: %v", mock.removedIDs)
	}
}

func TestCleanAutoNamedContainersAggregatesErrors(t *testing.T) {
	ctx := context.Background()
	mock := &mockDockerClient{
		containers: []container.Summary{
			{ID: "id1", Names: []string{"/inspiring_franklin"}},
			{ID: "id3", Names: []string{"/agitated_wescoff7"}},
		},
		removeErr: map[string]error{
			"id1": errors.New("permission denied"),
		},
	}

	removed, err := CleanAutoNamedContainers(ctx, mock)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if len(removed) != 1 || removed[0] != "agitated_wescoff7" {
		t.Fatalf("expected successful removal of id3, got %v", removed)
	}

	if !containsID(mock.removedIDs, "id1") || !containsID(mock.removedIDs, "id3") {
		t.Fatalf("expected removal attempts for both containers, got %v", mock.removedIDs)
	}
}

func containsID(ids []string, target string) bool {
	return slices.Contains(ids, target)
}
