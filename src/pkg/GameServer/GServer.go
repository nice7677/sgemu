package GameServer

import (
	"Core"
	"Data"
	. "SG"
	"net"
	"net/rpc"
	"strconv"
)

type GamePacketFunc func(c *GClient, p *SGPacket)

var (
	Handler map[int]GamePacketFunc

	Server *GServer
)

type GServer struct {
	Core.CoreServer
	WANAddr *net.TCPAddr
	WANIP   string
	Maps    map[int]*Map

	IDG   *Core.IDGen
	Run   *Core.Runner
	DBRun *Core.Runner
	Sdr   *Core.Scheduler

	RPCClient *rpc.Client
	RPCServer *rpc.Server
	RPCListener net.Listener
}

func (serv *GServer) OnSetup() {
	serv.CoreServer.OnSetup()
	serv.Maps = make(map[int]*Map)
	serv.IDG = Core.NewIDG()
	serv.Maps[0] = NewMap()
	serv.Run = Core.NewRunner()
	serv.DBRun = Core.NewRunner()
	serv.Sdr = Core.NewScheduler()
	serv.Run.Start()
	serv.DBRun.Start()
	serv.Sdr.Start()

	serv.Sdr.AddMin(func() { serv.SavePlayers() }, 1)
	 
	startRPCServer()
	
	go serv.AcceptClients()
	//serv.Sdr.AddSec(func() { serv.SavePlayers() }, 5)
}

func init() {
	Handler = map[int]GamePacketFunc{
		CSM_CHAT:         OnChat,
		CM_PING:          OnPing,
		CSM_MOVE:         OnMove,
		CM_PROFILE:       OnProfileRequest,
		CM_LEAVE_PROFILE: OnProfileLeave,
		CM_SHOP_REQUEST:  OnShopRequest,
		CSM_GAME_ENTER:   OnGameEnter,
		CM_DISCONNECT:    OnDisconnectPacket,
		CSM_LAB_ENTER:	  OnLabraryEnter,
	}
}

func (serv *GServer) Start(name, ip string, port int, wanip string) (err error) {
	err = Core.Start(serv, name, ip, port)
	if err != nil {
		return err
	}

	serv.WANIP = wanip
	serv.WANAddr, err = net.ResolveTCPAddr("tcp", serv.WANIP+":"+strconv.Itoa(port))
	if err != nil {
		serv.Log.Printf("Server start failed %s", err.Error())
		return err
	}

	return nil
}

func (serv *GServer) SavePlayers() {
	serv.Log.Printf("Saving Players...")
	for _, m := range serv.Maps {
		m.Run.Add(func() {
			for _, c := range m.Players {
				serv.DBRun.Add(func() { Data.SavePlayer(c.Player) })
			}
		})
	}
	serv.Sdr.AddMin(func() { serv.SavePlayers() }, 1)
	//serv.Sdr.AddSec(func() { serv.SavePlayers() }, 5)
}

func (serv *GServer) OnShutdown() {
	serv.Run.StopAndWait()
	serv.Log.Printf("GServer runner stopped!")
	serv.DBRun.StopAndWait()
	serv.Log.Printf("GServer DB runner stopped!")
	for id, m := range serv.Maps {
		m.Run.StopAndWait()
		serv.Log.Printf("Mapid %d runner stopped!", id)
		for _, c := range m.Players {
			Data.SavePlayer(c.Player)
		}
	}
	serv.CoreServer.Socket.Close()
	serv.Log.Printf("GServer socket closed!")
	
	if serv.RPCListener != nil {
		e := serv.RPCListener.Close()
		if e != nil  {
			panic(e)
		}
	}
}

func (serv *GServer) OnConnect(socket *net.TCPConn) {
	serv.Log.Printf("Client connected to GServer!")
	client := new(GClient)
	client.Server = serv
	Core.SetupClient(client, socket, serv)
}
