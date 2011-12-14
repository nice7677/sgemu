package Data

type Unit struct {
	UnitDB
	ID   uint32  
	Owner *Player
	Data *UnitData
}

type UnitDB struct {
	DBID  uint32 "_id"
	Level byte
	HP    uint32
	XP 	  uint32
	Squad string
	
}

func (u *Unit) TotalXP() uint32{
	n := uint32(u.Level+1)
	return n*n
}

//Unit Quality
func (u *Unit) UQ() byte{
	return u.Level + (u.Owner.Clout/2)
}

func (u *Unit) MaxWeight() uint16{
	return uint16(u.Data.Max_Weight) + (u.Data.Max_Weight * uint16(u.Owner.MechApt/120))
}

//Alien Tech Level
func (u *Unit) ATL() byte{
	return u.Level + ((u.Owner.MechApt + u.Owner.Education)/2)
}

//Tech Level
func (u *Unit) TL() byte{
	return u.Level + (u.Owner.Education/2)
}