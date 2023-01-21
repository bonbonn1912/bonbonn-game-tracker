import express from 'express'
import { Request, Response } from 'express'
import bodyParser from 'body-parser'
import path from 'path'
import cors from 'cors'

import { SECRETS } from './config/env'

const server = express()

server.use(express.static(path.join(__dirname, "frontend")));
server.use(express.json())
server.use(cors())
server.use(bodyParser.urlencoded({extended: true}))


server.get("/test", (req: Request , res: Response) =>{
    res.json({"message": SECRETS.env})
})

server.get("/*", (req: Request, res: Response) =>{
    res.sendFile(path.resolve(__dirname,"frontend", "index.html"))
})

console.log(SECRETS.var1)
server.listen(3000, () =>{console.log("Server listening on port 3000")})


