package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSetUsers(t *testing.T) {
	channelStore := NewChannelStore()
	users := []string{"a", "b"}
	channelStore.SetUsers(users, "srv", "#chan")
	assert.Equal(t, users, channelStore.GetUsers("srv", "#chan"))
	channelStore.SetUsers(users, "srv", "#chan")
	assert.Equal(t, users, channelStore.GetUsers("srv", "#chan"))
}

func TestAddRemoveUser(t *testing.T) {
	channelStore := NewChannelStore()
	channelStore.AddUser("user", "srv", "#chan")
	channelStore.AddUser("user", "srv", "#chan")
	assert.Len(t, channelStore.GetUsers("srv", "#chan"), 1)
	channelStore.AddUser("user2", "srv", "#chan")
	assert.Equal(t, []string{"user", "user2"}, channelStore.GetUsers("srv", "#chan"))
	channelStore.RemoveUser("user", "srv", "#chan")
	assert.Equal(t, []string{"user2"}, channelStore.GetUsers("srv", "#chan"))
}

func TestRemoveUserAll(t *testing.T) {
	channelStore := NewChannelStore()
	channelStore.AddUser("user", "srv", "#chan1")
	channelStore.AddUser("user", "srv", "#chan2")
	channelStore.RemoveUserAll("user", "srv")
	assert.Empty(t, channelStore.GetUsers("srv", "#chan1"))
	assert.Empty(t, channelStore.GetUsers("srv", "#chan2"))
}

func TestRenameUser(t *testing.T) {
	channelStore := NewChannelStore()
	channelStore.AddUser("user", "srv", "#chan1")
	channelStore.AddUser("user", "srv", "#chan2")
	channelStore.RenameUser("user", "new", "srv")
	assert.Equal(t, []string{"new"}, channelStore.GetUsers("srv", "#chan1"))
	assert.Equal(t, []string{"new"}, channelStore.GetUsers("srv", "#chan2"))

	channelStore.AddUser("@gotop", "srv", "#chan3")
	channelStore.RenameUser("gotop", "stillgotit", "srv")
	assert.Equal(t, []string{"@stillgotit"}, channelStore.GetUsers("srv", "#chan3"))
}

func TestMode(t *testing.T) {
	channelStore := NewChannelStore()
	channelStore.AddUser("+user", "srv", "#chan")
	channelStore.SetMode("srv", "#chan", "user", "o", "v")
	assert.Equal(t, []string{"@user"}, channelStore.GetUsers("srv", "#chan"))
	channelStore.SetMode("srv", "#chan", "user", "v", "")
	assert.Equal(t, []string{"@user"}, channelStore.GetUsers("srv", "#chan"))
	channelStore.SetMode("srv", "#chan", "user", "", "o")
	assert.Equal(t, []string{"+user"}, channelStore.GetUsers("srv", "#chan"))
	channelStore.SetMode("srv", "#chan", "user", "q", "")
	assert.Equal(t, []string{"~user"}, channelStore.GetUsers("srv", "#chan"))
}

func TestTopic(t *testing.T) {
	channelStore := NewChannelStore()
	assert.Equal(t, "", channelStore.GetTopic("srv", "#chan"))
	channelStore.SetTopic("the topic", "srv", "#chan")
	assert.Equal(t, "the topic", channelStore.GetTopic("srv", "#chan"))
}

func TestChannelUserMode(t *testing.T) {
	user := NewChannelStoreUser("&test")
	assert.Equal(t, "test", user.nick)
	assert.Equal(t, "a", string(user.modes[0]))
	assert.Equal(t, "&test", user.String())

	user.removeModes("a")
	assert.Equal(t, "test", user.String())
	user.addModes("o")
	assert.Equal(t, "@test", user.String())
	user.addModes("q")
	assert.Equal(t, "~test", user.String())
	user.addModes("v")
	assert.Equal(t, "~test", user.String())
	user.removeModes("qo")
	assert.Equal(t, "+test", user.String())
	user.removeModes("v")
	assert.Equal(t, "test", user.String())
}
