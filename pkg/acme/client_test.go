package acme

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, &ClientTestSuite{})
}

func (s *ClientTestSuite) TestObtainSSL() {
	client, err := NewRegisterClient("ci@haozi.net", "https://acme-staging-v02.api.letsencrypt.org/directory", KeyEC256)
	s.Nil(err)

	err = client.UseDns(DnsPod, DNSParam{
		ID:    "368120",
		Token: "5a0a7f5bedf840b9e061982b161489d7",
	})
	s.Nil(err)

	/*err = client.UseManualDns(false)
	s.Nil(err)

	resolves, err := client.GetDNSResolve([]string{"haozi.dev"})
	s.Nil(err)
	s.NotNil(resolves)*/

	ssl, err := client.ObtainSSL([]string{"hzbk.net"})
	s.Error(err)
	s.NotNil(ssl)
}
