package main

import (
	chat_api "chat_api"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
)

var Users = make(map[string]UserData, 10)

type UserData struct {
	chat_api.UserData `json:"data"`
	UserName          string `json:"username"`
	UserID            int64  `json:"user_id"`
}

var ErrUserFileNotExists = errors.New("user file not exists")

func LoadUserData(path string) (data *UserData, err error) {
	file, err_file := os.Open(path)

	if err_file != nil {
		err = fmt.Errorf("%w: %w", ErrUserFileNotExists, err_file)
		return
	}

	file_data, err_read := io.ReadAll(file)

	if err_read != nil {
		err = fmt.Errorf("unable to read userdata from file %w", err_read)
		return
	}

	err = json.Unmarshal(file_data, &data)

	if err != nil {
		err = fmt.Errorf("unable to load userdata from json %w", err)
		return
	}

	return

}

func LoadNamedUser(name string, id int64) (*UserData, error) {
	file_path := path.Join(UserDataFolder, name)

	data, err := LoadUserData(file_path)
	if errors.Is(err, ErrUserFileNotExists) {
		return NewUser(name, id)
	}

	return data, err
}

func (d *UserData) Store(path string) error {
	marshalled_json, err := json.Marshal(*d)

	if err != nil {
		return fmt.Errorf("failed to marshall userdata %w", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %w", err)
		}
	}

	err = os.WriteFile(path, marshalled_json, fs.ModePerm)

	if err != nil {
		return fmt.Errorf("failed to write data to file %w", err)
	}

	return nil
}

const UserDataFolder = "users"

var ErrFolderNotExists = errors.New("unable to create folder")
var ErrBlankUserName = errors.New("blank user name")

func (d *UserData) StoreSimple() error {

	if d.UserName == "" {
		return fmt.Errorf("trying to store data for unknown user %w", ErrBlankUserName)
	}

	if _, err := os.Stat(UserDataFolder); os.IsNotExist(err) {
		err := os.Mkdir(UserDataFolder, 0755)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFolderNotExists, UserDataFolder)
		}

	}

	file := path.Join(UserDataFolder, d.UserName)

	err := d.Store(file)

	if err != nil {
		return fmt.Errorf("unable to store user data %w", err)
	}

	return nil

}

func GetUser(name string, id int64) (result *UserData, err error) {
	user, ok := Users[name]
	if ok {
		result = &user
		return
	}

	result, err = LoadNamedUser(name, id)
	if err != nil {
		err = fmt.Errorf("failed to load data for user: %s; %w", name, err)
		return
	}

	return
}

func NewUser(name string, id int64) (result *UserData, err_back error) {
	data, err := chat_api.NewUser(chat_api.ModeChat)
	if err != nil {
		err_back = fmt.Errorf("unable to create newuser %w", err)
		return
	}

	result = &UserData{
		UserName: name,
		UserID:   id,
		UserData: data,
	}

	Users[name] = *result
	result.StoreSimple()

	return
}
