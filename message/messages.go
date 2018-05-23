package message

import (
	"github.com/rhymond/go-money"
	uuid "github.com/satori/go.uuid"
)

const (
	// KindConnected is sent when user connects
	KindConnected = iota + 1
	// KindUserJoined is sent when someone else joins
	KindUserJoined
	// KindUserLeft is sent when someone leaves
	KindUserLeft
	// KindTrack
	KindTrack
	// KindVehicle
	KindVehicle
	// KindCharger
	KindCharger
	// KindClear message is sent when a user clears the screen
	KindClear
	// KindTransaction
	KindTransaction
)

type User struct {
	ID    string `json:"id"`
	Color string `json:"color"`
}

type Connected struct {
	Kind  int    `json:"kind"`
	Color string `json:"color"`
	Users []User `json:"users"`
}

func NewConnected(color string, users []User) *Connected {
	return &Connected{
		Kind:  KindConnected,
		Color: color,
		Users: users,
	}
}

type UserJoined struct {
	Kind int  `json:"kind"`
	User User `json:"user"`
}

func NewUserJoined(userID string, color string) *UserJoined {
	return &UserJoined{
		Kind: KindUserJoined,
		User: User{ID: userID, Color: color},
	}
}

type UserLeft struct {
	Kind   int    `json:"kind"`
	UserID string `json:"userId"`
}

func NewUserLeft(userID string) *UserLeft {
	return &UserLeft{
		Kind:   KindUserLeft,
		UserID: userID,
	}
}

type Clear struct {
	Kind   int    `json:"kind"`
	UserID string `json:"userId"`
}

type Transaction struct {
	Kind   int          `json:"kind"`
	Id     string       `json:"id"`
	Units  int          `json:"units"`
	Amount money.Amount `json:"amount"`
}

func NewTransaction(units int, amount money.Amount) *Transaction {
	return &Transaction{
		Kind:   KindTransaction,
		Id:     uuid.Must(uuid.NewV4()).String(),
		Amount: amount,
	}
}
