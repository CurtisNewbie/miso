package testdata

import "github.com/curtisnewbie/miso/miso"

type CreateUserReq struct {
	Name string `json:"name"`
}

type CreateUserRes struct {
	UserID string `json:"userId"`
}

func CreateUser(inb *miso.Inbound, req CreateUserReq) (CreateUserRes, error) {
	return CreateUserRes{UserID: "123"}, nil
}

func init() {
	miso.HttpPost("/user", miso.AutoHandler(CreateUser)).
		Desc("Create user")
}
