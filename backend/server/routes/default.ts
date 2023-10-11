import { Router, type Request, type Response } from 'express'
import path from 'path'

const defaultRouter: Router = Router()

defaultRouter.get('/test', (req: Request, res: Response) => {
  res.send({ message: 'Hello World!' })
})

defaultRouter.post('/test', (req: Request, res: Response) => {
  res.send({ message: 'Hello World!' })
})

/**
 * ts-node needs another static path since the application doesnt run from build (frontend needs to be build before)
 */


export default defaultRouter
