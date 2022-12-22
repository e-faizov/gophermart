package models

const UserUUID = "user_uuid"

type ContextKey string

const UUIDKey ContextKey = UserUUID

type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
