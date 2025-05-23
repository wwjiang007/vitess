/*
Copyright 2019 The Vitess Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package servenv

import (
	"context"
	"strings"

	"github.com/spf13/pflag"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"vitess.io/vitess/go/vt/log"
	"vitess.io/vitess/go/vt/utils"
)

var (
	// clientCertSubstrings list of substrings of at least one of the client certificate names to use during authorization
	clientCertSubstrings string
	// MtlsAuthPlugin implements AuthPlugin interface
	_ Authenticator = (*MtlsAuthPlugin)(nil)
)

func registerGRPCServerAuthMTLSFlags(fs *pflag.FlagSet) {
	utils.SetFlagStringVar(fs, &clientCertSubstrings, "grpc-auth-mtls-allowed-substrings", clientCertSubstrings, "List of substrings of at least one of the client certificate names (separated by colon).")
}

// MtlsAuthPlugin  implements static username/password authentication for grpc. It contains an array of username/passwords
// that will be authorized to connect to the grpc server.
type MtlsAuthPlugin struct {
	clientCertSubstrings []string
}

// Authenticate implements Authenticator interface. This method will be used inside a middleware in grpc_server to authenticate
// incoming requests.
func (ma *MtlsAuthPlugin) Authenticate(ctx context.Context, fullMethod string) (context.Context, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "no peer connection info")
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "not connected via TLS")
	}
	for _, substring := range ma.clientCertSubstrings {
		for _, cert := range tlsInfo.State.PeerCertificates {
			if strings.Contains(cert.Subject.String(), substring) {
				return ctx, nil
			}
		}
	}
	return nil, status.Errorf(codes.Unauthenticated, "client certificate not authorized")
}

func mtlsAuthPluginInitializer() (Authenticator, error) {
	mtlsAuthPlugin := &MtlsAuthPlugin{
		clientCertSubstrings: strings.Split(clientCertSubstrings, ":"),
	}
	log.Infof("mtls auth plugin have initialized successfully with allowed client cert name substrings of %v", clientCertSubstrings)
	return mtlsAuthPlugin, nil
}

// ClientCertSubstrings returns the value of the
// `--grpc-auth-mtls-allowed-substrings` flag.
func ClientCertSubstrings() string {
	return clientCertSubstrings
}

func init() {
	RegisterAuthPlugin("mtls", mtlsAuthPluginInitializer)
	grpcAuthServerFlagHooks = append(grpcAuthServerFlagHooks, registerGRPCServerAuthMTLSFlags)
}
