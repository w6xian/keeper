package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"github.com/w6xian/sloth"
	"github.com/w6xian/sloth/nrpc/wsocket"
)

var (
	appPort string
	appPath string
)

var AppCmd = &cobra.Command{
	Use:   "app",
	Short: "Start the application logic",
	Run: func(cmd *cobra.Command, args []string) {
		runApp()
	},
}

func init() {
	AppCmd.Flags().StringVar(&appPort, "port", "", "Address of the keeper websocket server")
	AppCmd.Flags().StringVar(&appPath, "path", "", "Path of the keeper websocket server")
}

func runApp() {
	fmt.Printf("[App] Starting application process... PID: %d\n", os.Getpid())
	fmt.Println("[App] appPort=", appPort)
	if appPort != "" {
		fmt.Printf("[App] Connecting to Keeper at %s%s\n", appPort, appPath)

		go func() {
			// Client logic container (ServerRpc handles client-side logic for outgoing requests)
			serverRpc := sloth.DefaultClient()
			// Connection manager
			cliConn := sloth.ClientConn(serverRpc)

			// Dial
			go cliConn.StartWebsocketClient(
				wsocket.WithClientHandle(&Handler{}),
				wsocket.WithClientUriPath(appPath),
				wsocket.WithClientServerUri(appPort),
			)

			time.Sleep(1 * time.Second)

			// Call RPC
			// Call(ctx context.Context, mtd string, arg ...any) ([]byte, error)
			i := 1
			for {
				resp, err := serverRpc.Call(context.Background(), "keeper.SayHello", fmt.Sprintf("hello iam app %d", i))
				if err != nil {
					fmt.Println("[App] RPC Call Failed:", err)
				} else {
					// The response is []byte, assuming string
					// Note: sloth uses an Encoder/Decoder. Default might be gob or json.
					// If the server returns string "Hello AppProcess", the []byte might contain serialization overhead if it's gob.
					// But for now let's just print it.
					fmt.Println("[App] RPC Response:", string(resp))
				}
				time.Sleep(5 * time.Second)
				i++
			}
		}()
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	fmt.Println("out")
}

// Handler handles client-side WebSocket events
type Handler struct {
	server *sloth.ServerRpc
}

// OnClose is called when connection is closed
func (h *Handler) OnClose(ctx context.Context, c *wsocket.LocalClient, ch *wsocket.WsChannelClient) error {
	fmt.Println("OnClose:", ch.UserId)
	return nil
}

// OnData handles received messages
func (h *Handler) OnData(ctx context.Context, c *wsocket.LocalClient, ch *wsocket.WsChannelClient, msgType int, message []byte) error {
	if msgType == websocket.TextMessage {
		fmt.Println("HandleMessage:", 1, string(message))
	}

	return nil
}

// OnError handles errors
func (h *Handler) OnError(ctx context.Context, c *wsocket.LocalClient, ch *wsocket.WsChannelClient, err error) error {
	fmt.Println("OnError:", err.Error())
	return nil
}

// OnOpen is called when connection is opened
func (h *Handler) OnOpen(ctx context.Context, c *wsocket.LocalClient, ch *wsocket.WsChannelClient) error {
	fmt.Println("OnOpen:", ch.UserId, h.server)
	// Example of sending an initial message or setting state
	// ch.UserId = 2
	// ch.RoomId = 1
	// h.server.Send(context.Background(), map[string]string{"user_id": "2", "room_id": "1"})
	return nil
}
