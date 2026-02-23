package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"keeper/internal/registry"

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

			// --- Registry Logic ---
			instanceID := fmt.Sprintf("app-%d", os.Getpid())
			serviceName := "app-service"

			// 1. Register
			fmt.Println("[App] Registering service...")
			regReq := registry.RegisterRequest{
				Instance: registry.ServiceInstance{
					ID:   instanceID,
					Name: serviceName,
					Host: "127.0.0.1",
					Port: 0, // Fake port for now
					Tags: []string{"v1", "test"},
				},
			}
			regRespBytes, err := serverRpc.Call(context.Background(), "registry.Register", regReq)
			if err != nil {
				fmt.Printf("[App] Register failed: %v\n", err)
			} else {
				fmt.Printf("[App] Register success: %s\n", string(regRespBytes))
			}

			// Start Heartbeat Loop
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					_, err := serverRpc.Call(context.Background(), "registry.Heartbeat", registry.HeartbeatRequest{
						ServiceName: serviceName,
						InstanceID:  instanceID,
					})
					if err != nil {
						fmt.Printf("[App] Heartbeat failed: %v\n", err)
						// Retry registration
						serverRpc.Call(context.Background(), "registry.Register", regReq)
					}
				}
			}()
			// ----------------------

			// Call RPC
			// Call(ctx context.Context, mtd string, arg ...any) ([]byte, error)
			i := 1
			for {
				// 1. Hello
				resp, err := serverRpc.Call(context.Background(), "keeper.SayHello", fmt.Sprintf("hello iam app %d", i))
				if err != nil {
					fmt.Println("[App] RPC Call Failed:", err)
				} else {
					fmt.Println("[App] RPC Response:", string(resp))
				}

				// 2. Log
				_, err = serverRpc.Call(context.Background(), "log.Info", fmt.Sprintf("App log info %d", i))
				if err != nil {
					fmt.Println("[App] Log Info Failed:", err)
				}
				// Only log error occasionally to avoid spam
				if i%10 == 0 {
					_, err = serverRpc.Call(context.Background(), "log.Error", fmt.Sprintf("App log error %d", i))
					if err != nil {
						fmt.Println("[App] Log Error Failed:", err)
					}
				}

				// 3. Script
				if i%5 == 0 {
					luaScript := fmt.Sprintf("print('Hello from Lua! i=%d')", i)
					_, err = serverRpc.Call(context.Background(), "script.Run", luaScript)
					if err != nil {
						fmt.Println("[App] Script Run Failed:", err)
					}
				}

				// 4. Discovery Test
				if i%5 == 0 {
					discResp, err := serverRpc.Call(context.Background(), "registry.Discovery", registry.DiscoveryRequest{
						ServiceName: serviceName,
					})
					if err == nil {
						fmt.Printf("[App] Discovery Result: %s\n", string(discResp))
					}
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
