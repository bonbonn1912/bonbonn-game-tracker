import express from 'express'
import bodyParser from 'body-parser'
import path from 'path'
import cors from 'cors'

import defaultRouter from './routes/default'
import faceitRouter  from './routes/faceit'

const server = express()

/**
 * ts-node needs another static path since the application doesnt run from build (frontend needs to be build before)
 */
let staticPath: string = process.env.ts_node != "true" 
? path.join(__dirname, "../frontend")
: path.join(__dirname, "../../build/frontend")


server.use(express.static(staticPath));
server.use(express.json())
server.use(cors())
server.use(bodyParser.urlencoded({extended: true}))

server.use(faceitRouter)
server.use(defaultRouter)


export default server
