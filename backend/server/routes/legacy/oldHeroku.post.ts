import { Router, type Request, type Response } from 'express'
import { addMatchroomToDB } from '../../../util/database/addToDatabase'

const legacyPostRouter: Router = Router()

legacyPostRouter.post('/match', (req: Request, res: Response) => {
  addMatchroomToDB(req.body)
  res.json({ message: 'received' })
})

export default legacyPostRouter
