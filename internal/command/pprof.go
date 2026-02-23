package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/w6xian/sloth"
	"github.com/w6xian/sloth/nrpc/wsocket"
)

type WsInfo struct {
	Addr string
	Path string
}

func ReadWsInfoFromFile(filename string) (*WsInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	ws := &WsInfo{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "addr":
			ws.Addr = value
		case "path":
			ws.Path = value
		}
	}

	if ws.Addr == "" || ws.Path == "" {
		return nil, fmt.Errorf("incomplete ws info in %s", filename)
	}

	return ws, nil
}

// 执行 keeper pprof 命令 反序列化sloth服务端的pprof信息
var PprofCmd = &cobra.Command{
	Use:   "pprof",
	Short: "Execute pprof command",
	Run: func(cmd *cobra.Command, args []string) {
		wsInfo, err := ReadWsInfoFromFile("keeper.ws")
		if err != nil {
			log.Fatalf("ReadWsInfoFromFile failed: %v", err)
		}

		serverRpc := sloth.DefaultClient()
		cliConn := sloth.ClientConn(serverRpc)

		go cliConn.StartWebsocketClient(
			wsocket.WithClientUriPath(wsInfo.Path),
			wsocket.WithClientServerUri(wsInfo.Addr),
		)

		time.Sleep(1 * time.Second)

		resp, err := serverRpc.Call(context.Background(), "pprof.Services")
		if err != nil {
			log.Fatalf("Call pprof.Services failed: %v", err)
		}

		var info map[string][]string
		if err := json.Unmarshal(resp, &info); err != nil {
			log.Fatalf("Unmarshal PprofInfo response failed: %v", err)
		}

		for name, values := range info {
			fmt.Printf("%s:\n", name)
			for _, v := range values {
				fmt.Printf("  %s\n", v)
			}
		}
	},
}
