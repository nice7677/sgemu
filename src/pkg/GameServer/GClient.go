package GameServer

import (
	C "Core"
	. "Core/SG"
	D "Data"
	"log"
	//R "reflect"
)

type GClient struct {
	C.Client
	Key    byte
	packet *SGPacket

	ID     uint32
	Player *D.Player
	Units  map[uint32]*D.Unit
	Server *GServer
	Map    *Map
}

func (client *GClient) StartRecive() {
	defer client.OnDisconnect()
	for {
		bl, err := client.Socket.Read(client.packet.Buffer[client.packet.Index:])
		if err != nil {
			return
		}

		client.packet.Index += bl

		for client.packet.Index > 2 {
			p := client.packet
			size := p.Index
			p.Index = 0
			if p.ReadByte() != 0xAA {
				client.Log().Printf("Wrong packet header")
				client.Log().Printf("% #X", p.Buffer[:size])
				return
			}
			l := int(p.ReadUInt16())
			p.Index = size
			if len(client.packet.Buffer) < l {
				client.packet.Resize(l)
			}

			if size >= l+3 {
				temp := client.packet.Buffer[:l+3]
				op := client.packet.Buffer[3]
				if op > 13 || (op > 1 && op < 5) || (op > 6 && op < 13) {
					var sumCheck bool
					temp, sumCheck = DecryptPacket(temp)
					if !sumCheck {
						client.Log().Println("Packet sum check failed!")
						return
					}
				} else {
					temp = temp[3:]
				}
				client.ParsePacket(NewPacketRef(temp))
				client.packet.Index = 0
				if size > l+3 {
					client.packet.Index = size - (l + 3)
					copy(client.packet.Buffer, client.packet.Buffer[l+3:size])
				} else {
					//keeping the user under 4048k use to save memory
					if cap(client.packet.Buffer) > 4048 {
						client.Buffer = make([]byte, 1024)
						client.packet = NewPacketRef(client.Buffer)
					}
				}
			} else {
				break
			}
		}
	}
}

func (client *GClient) OnConnect() {

	userID, q := D.LoginQueue.Check(client.IP)
	if !q {
		client.OnDisconnect()
		return
	}

	id, r := client.Server.IDG.Next()

	if !r {
		client.OnDisconnect()
		return
	}

	client.Log().Println("ID " + userID)
	client.Player = D.GetPlayerByUserID(userID)

	client.Units = make(map[uint32]*D.Unit)

	for _, unitdb := range client.Player.UnitsData {
		id, r := client.Server.IDG.Next()
		if !r {
			client.OnDisconnect()
			return
		}
		name, e := D.Units[unitdb.Name]
		if !e {
			client.Log().Println("Unit name does not exists")
			continue
		}
		client.Units[id] = &D.Unit{unitdb, id, client.Player, name}
	}

	client.Log().Println("name " + client.Player.Name)

	client.packet = NewPacketRef(client.Buffer)
	client.packet.Index = 0
	client.ID = id

	Server.Run.Funcs <- func() {
		client.Server.Maps[0].OnPlayerJoin(client)
		client.SendWelcome()
	}
	client.StartRecive()
}

func (client *GClient) OnDisconnect() {
	if x := recover(); x != nil {
		client.Log().Printf("panic : %v \n %s", x, C.PanicPath())
	} 
	if client.Map != nil {
		client.Map.OnLeave(client)
	}
	if client.Units != nil {
		for id,_ := range client.Units {
			client.Server.IDG.Return(id)
		}
	}
	if client.Player != nil {
		client.Server.IDG.Return(client.ID)
		client.Server.DBRun.Funcs <- func() { D.SavePlayer(client.Player) }
	}
	client.Socket.Close()
	client.MainServer.GetServer().Log.Println("Client Disconnected!")
}

func (client *GClient) Send(p *SGPacket) {
	if !p.Encrypted {
		op := p.Buffer[3]
		if op > 13 || (op > 0 && op < 3) || (op > 3 && op < 11) {
			p.WSkip(2)
			EncryptPacket(p.Buffer[:p.Index], client.Key)
			p.Encrypted = true
			client.Key++
		}
		p.WriteLen()
	}
	client.Socket.Write(p.Buffer[:p.Index])
}

func (client *GClient) SendRaw(p *SGPacket) {
	p.WriteLen()
	client.Socket.Write(p.Buffer[:p.Index])
}
 
func (client *GClient) SendWelcome() {

	//player stats
	packet := NewPacket2(77)
	packet.WriteHeader(0x15)
	packet.WriteUInt32(client.ID)
	packet.WriteUInt32(12)
	packet.WriteUInt32(12)
	packet.WriteByte(9)
	packet.WriteUInt32(0)

	packet.WriteInt32(client.Player.Money)
	packet.WriteInt32(client.Player.Ore)
	packet.WriteInt32(client.Player.Silicon)
	packet.WriteInt32(client.Player.Uranium)
	packet.WriteByte(client.Player.Sulfur)
	packet.WriteInt32(6)
	packet.WriteByte(client.Player.Tactics)
	packet.WriteByte(client.Player.Clout)
	packet.WriteByte(client.Player.Education)
	packet.WriteByte(client.Player.MechApt)
	packet.Write([]byte{
		0x30, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x01, 0x00, 0x00, 0x01, 0x19, 0x00})
	//packet.Write([]byte{0x00, 0x00, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x0C, 0x07, 0x00, 0x00, 0x00, 0x01, 0x00, 0x04, 0x95, 0xD4, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x0F, 0x0A, 0x0A, 0x05, 0x30, 0x00, 0x00, 0x00, 0x02, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x64, 0x00, 0x01, 0x00, 0x00, 0x01, 0x19, 0x00})
	client.Send(packet)

	packet = NewPacket2(10 + len(client.Units)*100)
	packet.WriteHeader(0x16)
	packet.WriteByte(1)
	packet.WriteByte(byte(len(client.Units)))
	client.Log().Print(len(client.Units))
	for id, unit := range client.Units {
		packet.WriteUInt32(id)
		packet.WriteUInt16(0x113) //unit id
		packet.WriteUInt16(2)
		packet.WriteUInt32(30)  //xp
		packet.WriteInt32(0)    //xp modifier
		packet.WriteUInt32(100) //xp total
		packet.WriteByte(unit.Level)
		packet.WriteByte(1)
		packet.WriteByte(1)
		packet.WriteUInt16(1570) //hp 
		packet.WriteUInt16(1570) //max hp
		packet.WriteUInt16(0x44) //max-weight?
		packet.WriteUInt16(8)    //space?
		packet.WriteUInt16(0x48) //weight?
		packet.WriteUInt16(8)    //space?
		packet.WriteUInt16(0x4b) //unit-weight? 
		packet.WriteUInt16(0x30) //speed *10
		packet.WriteUInt16(0x12c)
		packet.WriteByte(1)
		packet.WriteUInt16(9) //armor?
		packet.WriteUInt16(0)
		packet.WriteUInt16(100)
		packet.WriteUInt16(0x62)  //fire power?
		packet.WriteUInt16(0x168) //range * 2 / 10
		packet.WriteUInt16(0xc8)  //cooldown * 100
		packet.WriteUInt16(0x62)  //fire power?
		packet.WriteUInt16(0x168) //range * 2 / 10
		packet.WriteUInt16(0xc8)  //cooldown * 100
		packet.WriteUInt64(0x9000006)
		packet.WriteUInt16(1) //kills
		packet.WriteString(unit.CustomName)
		packet.WriteString(unit.Name)
	}
	client.Send(packet)

	for id, unit := range client.Units {
		packet = NewPacket2(50)
		packet.WriteHeader(0x1F)
		packet.WriteUInt32(id)

		i := packet.Index
		packet.WriteByte(0)

		mi := byte(0)
		for _, item := range unit.Items {
			if item != nil {
				packet.WriteUInt16(item.ID)
				mi++
			}
		}

		packet.Buffer[i] = mi

		packet.WriteByte(0)
		client.Send(packet)
	}

	packet = NewPacket2(50)
	packet.WriteHeader(0x1F)
	packet.WriteUInt32(client.ID)
	packet.WriteByte(byte(len(client.Player.Items)))

	for _, item := range client.Player.Items {
		if item != nil {
			packet.WriteUInt16(item.ID)
		}
	}
	packet.WriteByte(0)
	client.Send(packet)

	//send map info
	packet = NewPacket2(198)
	packet.WriteHeader(0x17)
	packet.Write([]byte{0x00, 0x01, 0x87, 0x0A, 0x01, 0x00, 0x00, 0x00, 0x0C, 0x00, 0x00, 0x7B, 0x48, 0x98, 0xE4, 0x7B, 0x49, 0xF8, 0x74, 0x00, 0x0D, 0x00}) //, 0x00, 0x00, 0x0D, 0x42, 0x01, 0xC0, 0x0C, 0x40, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x41, 0x0C, 0x00, 0x0B, 0x40, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x40, 0x04, 0x00, 0x08, 0x80, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x3F, 0x0A, 0x60, 0x07, 0xA0, 0x00, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x3E, 0x03, 0xA0, 0x02, 0x80, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x3D, 0x0B, 0xE0, 0x02, 0x40, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x3C, 0x07, 0x80, 0x01, 0x60, 0x1D, 0x4C, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00, 0x00, 0x0D, 0x3B, 0x06, 0x20, 0x04, 0x40, 0x1D, 0x4C, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00, 0x00, 0x0D, 0x3A, 0x07, 0x40, 0x08, 0x80, 0x1D, 0x4C, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00, 0x00, 0x0D, 0x39, 0x09, 0x40, 0x0C, 0x00, 0x1D, 0x4C, 0x00, 0x00, 0x00, 0x00, 0x0D, 0x00, 0x00, 0x0D, 0x38, 0x07, 0xC0, 0x0E, 0xA0, 0x1D, 0x4C, 0x00, 0x00, 0x00, 0x00, 0x0D})
	client.Send(packet)

	//packet = C.NewPacket2(243)
	//packet.WriteHeader(0x17)
	//packet.Write([]byte{0x00, 0x01, 0x87, 0x0B, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x00, 0x01, 0x69, 0x28, 0x5B, 0xE8, 0x69, 0x29, 0xBB, 0x78, 0x00, 0x0C, 0x0E, 0x00, 0x00, 0x0C, 0x56, 0x0D, 0x40, 0x0D, 0xA0, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x55, 0x08, 0x20, 0x0C, 0x60, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x54, 0x03, 0xA0, 0x0B, 0x80, 0x00, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x53, 0x0A, 0x60, 0x09, 0xC0, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x52, 0x05, 0xE0, 0x09, 0xC0, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x51, 0x04, 0x20, 0x04, 0xC0, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x50, 0x0C, 0xC0, 0x04, 0x40, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x4F, 0x08, 0xC0, 0x03, 0xC0, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x4E, 0x02, 0xA0, 0x02, 0x80, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x4D, 0x0D, 0x60, 0x02, 0x00, 0x00, 0x00, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x4C, 0x08, 0xC0, 0x05, 0x60, 0x17, 0x70, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x4B, 0x05, 0x60, 0x07, 0xC0, 0x17, 0x70, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x4A, 0x0B, 0x40, 0x08, 0x00, 0x17, 0x70, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x08, 0x80, 0x0A, 0xA0, 0x17, 0x70, 0x00, 0x00, 0x00, 0x00, 0x00})
	//client.Send(packet)

	//send player name 
	packet = NewPacket2(28 + len(client.Map.Players)*13)
	packet.WriteHeader(0x47)
	packet.WriteInt16(int16(len(client.Map.Players)))
	for _, s := range client.Map.Players {
		packet.WriteString(s.Player.Name)
		packet.WSkip(2)
	}
	client.Send(packet)

	//send spawn palyer
	client.Map.OnPlayerAppear(client)

	packet = NewPacket2(18)
	packet.WriteHeader(0x0E)
	packet.Write([]byte{0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	client.Send(packet)

	packet = NewPacket2(13)
	packet.WriteHeader(0x0E)
	packet.Write([]byte{0x05, 0x00})
	client.Send(packet)

	packet = NewPacket2(13)
	packet.WriteHeader(0x3E)
	packet.Write([]byte{0x00, 0x00})
	//client.Map.Send(packet)
}

func (client *GClient) Log() *log.Logger {
	return Server.Log
}

func (client *GClient) ParsePacket(p *SGPacket) {
	header := p.ReadByte()

	fnc, exist := Handler[int(header)]
	if !exist {
		client.Log().Printf("isnt registred : %s", p)
		return
	}
	//client.Log().Printf("Header(%d) len(%d) : % #X\n %s", header, len(p.Buffer), p.Buffer, p.Buffer)
	//client.Log().Printf("Handle %s\n", R.TypeOf(fnc))

	fnc(client, p)
}
