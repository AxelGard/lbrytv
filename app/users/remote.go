package users

import (
	"github.com/lbryio/lbry.go/v2/extras/lbryinc"
	"github.com/lbryio/lbrytv/config"
)

// RemoteUser encapsulates internal-apis user data
type RemoteUser struct {
	ID    int
	Email string
}

func getRemoteUser(token string, remoteIP string) (*RemoteUser, error) {
	u := &RemoteUser{}
	c := lbryinc.NewClient(token, &lbryinc.ClientOpts{
		ServerAddress: config.GetInternalAPIHost(),
		RemoteIP:      remoteIP,
	})

	r, err := c.UserMe()
	if err != nil {
		// Conn.Logger.LogF(monitor.F{monitor.TokenF: token}).Error("internal-api responded with an error: ", err)
		// No user found in internal-apis database, give up at this point
		return nil, err
	}
	u.ID = int(r["id"].(float64))
	if r["primary_email"] != nil {
		u.Email = r["primary_email"].(string)
	}
	return u, nil
}
