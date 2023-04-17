import express from 'express'
import bodyParser from 'body-parser'
import path from 'path'
import cors from 'cors'

import defaultRouter from './routes/default'
import legacyGetRouter from './routes/legacy/oldHeroku.get'
import legacyPostRouter from './routes/legacy/oldHeroku.post'
import consolePostRouter from './routes/legacy/consoleelo.post'

const server = express()

/**
 * ts-node needs another static path since the application doesnt run from build (frontend needs to be build before)
 */
const staticPath: string = process.env.ts_node != 'true'
  ? path.join(__dirname, '../frontend')
  : path.join(__dirname, '../../build/frontend')

server.use(express.static(staticPath))
server.use(express.json())
server.use(cors())
server.use(bodyParser.urlencoded({ extended: true }))

server.use(consolePostRouter)
server.use(legacyGetRouter)
server.use(legacyPostRouter)

server.use(defaultRouter)

export default server
