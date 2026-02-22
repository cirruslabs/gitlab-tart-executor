package localnetworkhelper

import (
	"context"
	"net"
	"runtime"

	"github.com/cirruslabs/chacha/pkg/localnetworkhelper"
	"github.com/cirruslabs/chacha/pkg/privdrop"
	dialerpkg "github.com/cirruslabs/gitlab-tart-executor/internal/dialer"
	"github.com/spf13/cobra"
)

var username string

func IntroduceFlag(command *cobra.Command) {
	// We only need privilege dropping on macOS due to newly introduced
	// "Local Network" permission, which cannot be disabled automatically,
	// and according to the Apple's documentation[1], running Persistent
	// Worker as a superuser is the only choice.
	//
	// Note that the documentation says that "macOS automatically allows
	// local network access by:" and "Any daemon started by launchd". However,
	// this is not true for daemons that have <key>UserName</key> set to non-root.
	//
	//nolint:lll // can't make the link shorter
	// [1]: https://developer.apple.com/documentation/technotes/tn3179-understanding-local-network-privacy#macOS-considerations
	if runtime.GOOS != "darwin" {
		return
	}

	command.Flags().StringVar(&username, "user", "", "username to drop privileges to "+
		"(\"Local Network\" permission workaround: requires starting \"gitlab-tart-executor\" as \"root\", "+
		"the privileges will be then dropped to the specified user after starting the "+
		"\"gitlab-tart-executor localnetworkhelper\" helper process)")
}

//nolint:ireturn // it's not possible to return a non-interface here
func ConnectAndDropPrivileges(ctx context.Context) (dialerpkg.Dialer, error) {
	// Run the macOS "Local Network" permission helper
	// when privilege dropping is requested
	var dialer dialerpkg.Dialer

	if username != "" {
		localNetworkHelper, err := localnetworkhelper.New(ctx)
		if err != nil {
			return nil, err
		}

		dialer = dialerpkg.DialFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
			return localNetworkHelper.PrivilegedDialContext(ctx, network, addr)
		})

		if err := privdrop.Drop(username); err != nil {
			return nil, err
		}
	} else {
		dialer = &net.Dialer{}
	}

	return dialer, nil
}
