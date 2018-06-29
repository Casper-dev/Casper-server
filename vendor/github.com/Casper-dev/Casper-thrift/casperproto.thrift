# Struct incorporating both node's resulting hash and address
struct HashAddr {
	1: string hash;
	2: string addr;
}

struct NodeInfo {
    1:string ipfsAddr
    2:string thriftAddr
}

struct ChunkInfo {
	1:string uuid;
	2:i64 first;
	3:i64 last;
	4:string initiator;
	5:list<NodeInfo> providers;
	6:string diffuse;
}

struct PingResult {
	1:i64 timestamp;
	2:string id;
}

service CasperServer {

/*
 * Casper's Proxy API RPC methods
 */
	string SendConnectQuery()
	
/*
 * Casper's p2p API RPC methods service
 * Called only from client to server
 * Consists of basic API methods: Upload, Download, Update, Delete
 */
    string SendUploadQuery(1:string hash, 2:string ipfsAddr, 3:i64 sizeToStore)
    string SendDownloadQuery(1:string hash, 2:string ipfsAddr, 3:string wallet)
    string SendUpdateQuery(1:string uuid, 2:string hash, 3:i64 sizeToStore) // uuid is base58 encoded
	string SendDeleteQuery(1:string hash)


/*
 * Casper's p2p network consistency control methods
 * Called only from server to server
 *
 */
#Methods used in data integrity verification
	///Initialization
	void SendVerificationQuery(1:string UUID, 2:NodeInfo ninfo)
	void SendChunkInfo(1:ChunkInfo info)

	///Round one methods
	void SendChecksumHash(1:string UUID, 2:string ipfsAddr, 3:string hashDiffuse)

    // addrToHash is a mapping: <NodeID> -> <result vector>
	void SendValidationResults(1:string UUID, 2:string ipfsAddr, 3:map<string,string> addrToHash)

	///Utility methods
	/*returns base58 encoded multihash of file[first, last)*/
	string GetFileChecksum(1:string uuid, 2:i64 first, 3:i64 last, 4:string salt)

#Methods used in replication logics
	string SendReplicationQuery(1:string hash, 2:string blockedIpfsAddr, 3:i64 sizeToStore)
#Methods used in network connection verification logics
	PingResult Ping()
}

