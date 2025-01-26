package brick

import "github.com/doraemonkeys/doraemon"

func RandomLiveID() string {
	return doraemon.GenRandomString("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", 25)
}
