package mock_utils

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
)

func NeedChainState(client *mock_grpcpb.MockApiServiceClient) {
	resp := &grpcpb.GetChainStateResponse{
		State: &grpcpb.ChainState{
			Dgpo: &prototype.DynamicProperties{
				HeadBlockId: &prototype.Sha256{ Hash: make([]byte, 32) },
				Time: &prototype.TimePointSec{},
			},
		},
	}
	client.EXPECT().GetChainState(gomock.Any(), gomock.Any()).Return(resp, nil).AnyTimes()
}
