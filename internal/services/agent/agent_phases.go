package agent

import (
	"container/list"
	"context"
	"errors"
	"path/filepath"

	"cds.agents.app/pkg"
	"github.com/openai/openai-go/v2"
	"golang.org/x/sync/errgroup"
)

// PlanPhases orders tool calls into sequential phases with intra-phase parallelism.
// Flow: called by Run() after extracting tool calls.
// Yields: none; returns phase layers for execution.
func (a *Agent) PlanPhases(root string, calls []pkg.ToolCallLite) ([][]int, error) {
	// Fill in normalized paths for planning
	for i := range calls {
		p, d := pkg.AnalyzeLite(root, calls[i].FuncName, calls[i].FuncArgs)
		calls[i].PathAbs, calls[i].DirAbs = p, d
	}

	adj := make([][]int, len(calls))
	indeg := make([]int, len(calls))
	addEdge := func(u, v int) { // u -> v
		adj[u] = append(adj[u], v)
		indeg[v]++
	}

	// Group by file path and by directory (for list_dir)
	byPath := map[string][]int{}
	byDirList := map[string][]int{}
	for i, c := range calls {
		if c.PathAbs != "" {
			byPath[c.PathAbs] = append(byPath[c.PathAbs], i) // preserves input order
		}
		if c.FuncName == "list_dir" && c.DirAbs != "" {
			byDirList[c.DirAbs] = append(byDirList[c.DirAbs], i)
		}
	}

	// Same-file ordering
	for _, idxs := range byPath {
		var writes, reads, deletes []int
		for _, i := range idxs {
			switch calls[i].FuncName {
			case "write_file":
				writes = append(writes, i)
			case "read_file":
				reads = append(reads, i)
			case "delete_path":
				deletes = append(deletes, i)
			}
		}

		// Deterministic ordering among multiple writes on the same file:
		for k := 0; k+1 < len(writes); k++ {
			addEdge(writes[k], writes[k+1])
		}

		// writes happen before reads on the same file
		for _, w := range writes {
			for _, r := range reads {
				addEdge(w, r)
			}
		}
		// everything (reads/writes) happens before delete on the same file
		for _, i := range idxs {
			for _, d := range deletes {
				if i != d {
					addEdge(i, d)
				}
			}
		}
	}

	// Directory listings should reflect writes/deletes in that directory
	for dir, listers := range byDirList {
		for i, c := range calls {
			if c.PathAbs == "" {
				continue
			}
			if filepath.Dir(c.PathAbs) != dir {
				continue
			}
			if c.FuncName == "write_file" || c.FuncName == "delete_path" {
				for _, ld := range listers {
					addEdge(i, ld)
				}
			}
		}
	}

	// Kahn's algorithm into phases
	var phases [][]int
	q := list.New()
	for i := range calls {
		if indeg[i] == 0 {
			q.PushBack(i)
		}
	}
	for q.Len() > 0 {
		var layer []int
		sz := q.Len()
		for k := 0; k < sz; k++ {
			e := q.Front()
			q.Remove(e)
			u := e.Value.(int)
			layer = append(layer, u)
			for _, v := range adj[u] {
				indeg[v]--
				if indeg[v] == 0 {
					q.PushBack(v)
				}
			}
		}
		phases = append(phases, layer)
	}

	// Detect cycles -> fallback to sequential phase in original order
	remaining := 0
	for _, d := range indeg {
		if d > 0 {
			remaining++
		}
	}
	if remaining > 0 {
		seq := make([]int, len(calls))
		for i := range calls {
			seq[i] = i
		}
		return [][]int{seq}, nil
	}
	return phases, nil
}

// RunPhases executes phases sequentially, tool calls concurrently per phase.
// Flow: invoked by Run() after planning.
// Yields: appends ToolMessage results for each call; no final user text here.
func (a *Agent) RunPhases(toolCalls []pkg.ToolCallLite, phases [][]int, msg openai.ChatCompletionMessage) error {

	// Collect results for each tool call index
	type toolResult struct{ id, out string }
	results := make([]toolResult, len(toolCalls))

	// ===================== PHASE EXECUTION =====================
	//
	// Mental model:
	// - We first plan an order of operations and group tool calls into phases.
	// - **Phases are blocking/sequential**: we wait for the whole phase to finish before starting the next.
	// - **Inside a phase**, every tool call without mutual dependencies runs **concurrently** (each in its own goroutine).
	//
	// Example:
	//   1) write a/b.txt
	//   2) read  a/b.txt
	//   3) list_dir a
	//   4) read  c/d.go
	//
	// Edges (must-happen-before): 12 and 13. No edge touches 4.
	// Phases become:
	//   Phase 1: run (1) and (4) in parallel
	//   Phase 2: after Phase 1 finishes, run (2) and (3) in parallel
	//
	// Per-path locks (RWMutex) + atomic writes make it safe inside a phase:
	//   - read_file / list_dir use RLock (shared)
	//   - write_file / delete_path use Lock (exclusive)
	// ===========================================================
	for _, layer := range phases {
		g, gctx := errgroup.WithContext(context.Background())
		sem := make(chan struct{}, a.Concurrency) // bounded worker pool for this phase

		for _, i := range layer {
			i := i // capture
			// tc := toolCalls[i]

			g.Go(func() error {
				// acquire a worker slot
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-gctx.Done():
					return gctx.Err()
				}

				// Run tool via original SDK messages ToolCalls (same index)
				raw := msg.ToolCalls[i].Function.Arguments
				name := msg.ToolCalls[i].Function.Name
				out, err := a.Tooling(a.Src, name, raw)
				if err != nil {
					out = "ERROR: " + err.Error()
				}
				results[i] = toolResult{id: msg.ToolCalls[i].ID, out: out}
				return nil
			})
		}
		//  blocking: wait for the whole phase to complete
		if err := g.Wait(); err != nil {
			return err
		}
	}
	// ===========================================================

	// Feed ALL tool results for this assistant turn back to the model.
	for i := range toolCalls {
		a.Params.Messages = append(a.Params.Messages, openai.ToolMessage(results[i].out, results[i].id))
	}
	return errors.New("stopped: exceeded max steps")
}
