namespace go im.rpc

struct Request {  
	1: string message
}

struct Response {
	1: string message
}


// 发送消息
struct SendMsgReq {
    1: string uid; // uid
    2: binary msg; // 消息
}

struct SendMsgRsp {
    1: i32 retCode;
}


// 发送消息到组内用户
struct SendMsgByTgidReq {
    1: string uid; // uid
    2: string tgid; // uid
    3: binary msg; // 消息
}


//uid
struct UserIdReq {
    1: string uid; // uid
}

struct StopUserClientRsp {
    1: bool res; // uid
}


struct BytesRsp {
    1: binary data;
}

//订阅临时组
struct JoinGroupReq {
    1: string uid; // uid
    2: string tgid;
}

struct JoinGroupRes {
    1: bool res;
}



service ImRpc {
    Response hello(1: Request req)
    SendMsgRsp SendMsgToUser(1: SendMsgReq req) // 发送消息到用户
	SendMsgRsp SendMsgToUserByTgid(1: SendMsgByTgidReq req) // 发送消息到组内用户
	StopUserClientRsp StopUserClient(1: UserIdReq req) // 停止一个用户
	BytesRsp GetSysInfo() // 获取服务系统信息
    JoinGroupRes JoinGroup(1: JoinGroupReq req) // 订阅临时组
    JoinGroupRes OutGroup(1: JoinGroupReq req) // 离开临时组
}

# 进入到service/api/server目录 执行 kitex -module github.com/lizuowang/gim  ../api.thrift

