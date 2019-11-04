package pbserver

import (
	"context"
	pb "github.com/alvistar/nanopb/nanoproto"
)

func (server *Server) AccountBalance(ctx context.Context, pbRequest *pb.AccountBalanceRequest) (*pb.AccountBalanceReply, error) {
	request, _ := getAction(pbRequest, "account_balance", nil)

	reply := pb.AccountBalanceReply{}

	if err := server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}
}

func (server *Server) AccountCreate(ctx context.Context, pbRequest *pb.AccountCreateRequest) (*pb.AccountCreateReply, error) {
	request, _ := getAction(pbRequest, "account_create", nil)

	reply := pb.AccountCreateReply{}

	if err := server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}
}

func (server *Server) ValidateAccountNumber(ctx context.Context, pbRequest *pb.ValidateAccountNumberRequest) (*pb.ValidateAccountNumberReply, error) {
	request, _ := getAction(pbRequest, "validate_account_number", nil)

	reply := pb.ValidateAccountNumberReply{}

	if err := server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}
}

func (server *Server) Send(ctx context.Context, pbRequest *pb.SendRequest) (*pb.SendReply, error) {
	request, _ := getAction(pbRequest, "send", nil)

	reply := pb.SendReply{}

	if err := server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}
}

func (server *Server) AccountsBalances(ctx context.Context, pbRequest *pb.AccountsBalancesRequest) (*pb.AccountsBalancesReply, error) {

	request, _ := getAction(pbRequest, "accounts_balances", nil)

	reply := pb.AccountsBalancesReply{}

	if err := server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}

}

func (server *Server) BlockInfo(ctx context.Context, pbRequest *pb.BlockInfoRequest) (*pb.BlockInfoReply, error) {
	transform := TransformOpt{
		"json_block": str("true"),
	}

	request, _ := getAction(pbRequest, "block_info",
		transform)

	reply := pb.BlockInfoReply{}

	if err := server.handler(request, &reply); err == nil {
		return &reply, nil
	} else {
		return nil, err
	}

}

func (server *Server) BlocksInfo(request *pb.BlocksInfoRequest, stream pb.Nano_BlocksInfoServer) error {
	for _, hash := range request.Hashes {
		bir := pb.BlockInfoRequest{Hash: hash}
		bip, err := server.BlockInfo(context.Background(), &bir)

		if err != nil {
			return err
		}
		
		reply := pb.BlocksInfoReply{
			BlockHash: hash,
			Block:     bip,
		}

		if err := stream.Send(&reply); err != nil {
			return err
		}
	}

	return nil
}

//func (server *Server) BlocksInfo(ctx context.Context, pbRequest *pb.BlocksInfoRequest) (*pb.BlocksInfoReply, error) {
//	transform := TransformOpt{
//		"json_block": str("true"),
//		"include_not_found": boolToStr(),
//	}
//
//	request, _ := getAction(pbRequest, "blocks_info",
//		transform)
//
//	reply := pb.BlocksInfoReply{}
//
//	if err := server.handler(request, &reply); err == nil {
//		return &reply, nil
//	} else {
//		return nil, err
//	}
//}
