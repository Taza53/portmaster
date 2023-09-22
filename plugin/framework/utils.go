package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/safing/portmaster/plugin/shared/decider"
	"github.com/safing/portmaster/plugin/shared/proto"
	"github.com/safing/portmaster/plugin/shared/reporter"
)

type (
	// DeciderFunc is a utility type to implement a decider.Decider using
	// a function only.
	//
	// It implements decider.Decider.
	DeciderFunc func(context.Context, *proto.Connection) (proto.Verdict, string, error)

	// ReporterFunc is a utility type to implement a reporter.Reporter using
	// a function only.
	//
	// It implements reporter.Reporter.
	ReporterFunc func(context.Context, *proto.Connection) error

	// ResolverFunc is a utility type to implement a resolver.Resolver using
	// a function only
	//
	// It implements resolver.Resolver
	ResolverFunc func(context.Context, *proto.DNSQuestion, *proto.Connection) (*proto.DNSResponse, error)
)

var (
	getExecPathOnce        sync.Once
	executablePath         string
	resolvedExecutablePath string
	errGetExecPath         error
)

// DecideOnConnection passes through to fn and implements decider.Decider.
func (fn DeciderFunc) DecideOnConnection(ctx context.Context, conn *proto.Connection) (proto.Verdict, string, error) {
	return fn(ctx, conn)
}

// ReportConnection passes through to fn and implements reporter.Reporter.
func (fn ReporterFunc) ReportConnection(ctx context.Context, conn *proto.Connection) error {
	return fn(ctx, conn)
}

// Resolve passes through to fn and implements resolver.Resolver.
func (fn ResolverFunc) Resolve(ctx context.Context, question *proto.DNSQuestion, conn *proto.Connection) (*proto.DNSResponse, error) {
	return fn(ctx, question, conn)
}

// ChainDeciders is a utility method to register more than on decider in a plugin.
// It executes the deciders one after another and returns the first error encountered
// or the first verdict that is not VERDICT_UNDECIDED, VERDICT_UNDERTERMINABLE or VERDICT_FAILED.
//
// If a decider returns a nil error but VERDICT_FAILED ChainDeciders will still return a non-nil
// error.
func ChainDeciders(deciders ...decider.Decider) DeciderFunc {
	return func(ctx context.Context, c *proto.Connection) (proto.Verdict, string, error) {
		for idx, d := range deciders {
			verdict, reason, err := d.DecideOnConnection(ctx, c)
			if err != nil {
				return verdict, reason, err
			}

			switch verdict {
			case proto.Verdict_VERDICT_UNDECIDED,
				proto.Verdict_VERDICT_UNDETERMINABLE:
				continue

			case proto.Verdict_VERDICT_FAILED:
				return verdict, reason, fmt.Errorf("chained decider at index %d return VERDICT_FAILED", idx)

			default:
				return verdict, reason, nil
			}
		}

		return proto.Verdict_VERDICT_UNDECIDED, "", nil
	}
}

// ChainDeciderFunc is like ChainDeciders but accepts DeciderFunc instead of decider.Decider.
// This is mainly for convenience to avoid casting to DeciderFunc for each parameter passed
// to ChainDecider.
func ChainDeciderFunc(fns ...DeciderFunc) DeciderFunc {
	deciders := make([]decider.Decider, len(fns))
	for idx, fn := range fns {
		deciders[idx] = fn
	}

	return ChainDeciders(deciders...)
}

func getExecPath() {
	getExecPathOnce.Do(func() {
		executablePath, errGetExecPath = os.Executable()
		if errGetExecPath != nil {
			return
		}

		resolvedExecutablePath, errGetExecPath = filepath.EvalSymlinks(executablePath)
	})
}

// AllowPluginConnections returns a decider function that will allow outgoing and
// incoming connections to the plugin itself.
// This is mainly used in combination with ChainDecider or ChainDeciderFunc.
//
//	framework.RegisterDecider(framework.ChainDeciderFunc(
//		AllowPluginConnections(),
//		yourDeciderFunc,
//	))
func AllowPluginConnections() DeciderFunc {
	return func(ctx context.Context, c *proto.Connection) (proto.Verdict, string, error) {
		self, err := IsSelf(c)
		if err != nil {
			return proto.Verdict_VERDICT_UNDECIDED, "", fmt.Errorf("failed to get executable path: %w", errGetExecPath)
		}
		if self {
			return proto.Verdict_VERDICT_ACCEPT, "own plugin connections are allowed", nil
		}

		return proto.Verdict_VERDICT_UNDECIDED, "", nil
	}
}

// IsSelf returns true if the connection is initiated by the plugin itself.
// This is useful to, for example, filter connection that are initiated by the plugin
// and would otherwise cause endless loops.
func IsSelf(conn *proto.Connection) (bool, error) {
	getExecPath()

	binary := conn.GetProcess().GetBinaryPath()

	if errGetExecPath != nil {
		return false, errGetExecPath
	}

	if binary == resolvedExecutablePath || binary == executablePath {
		return true, nil
	}

	return false, nil
}

var (
	_ decider.Decider   = new(DeciderFunc)
	_ reporter.Reporter = new(ReporterFunc)
)
