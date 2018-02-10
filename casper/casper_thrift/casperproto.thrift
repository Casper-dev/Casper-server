service CasperServer{
    string SendUploadQuery(1:string hash, 2:string ipfsAddr, 3:i64 sizeToStore)
    string SendDownloadQuery(1:string hash, 2:string ipfsAddr, 3:string wallet)
    string SendDeleteQuery(1:string hash)
    i64 Ping()
}