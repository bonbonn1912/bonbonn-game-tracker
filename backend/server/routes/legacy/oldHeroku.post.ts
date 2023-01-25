import { Router, type Request, type Response } from 'express'
import { addMatchroomToDB } from '../../../util/database/mongo'
import { addEloInformation, validateAuthorizationHeader, validateEvent, validateStreamerGame } from '../../middleware/match'
import { addGame, getGame } from '../../../util/liveGames'
import { isTest } from '../../../config/env'

const legacyPostRouter: Router = Router()

legacyPostRouter.post('/match',validateAuthorizationHeader,validateEvent, validateStreamerGame, addEloInformation, (req: Request, res: Response) => {
  addGame(req.headers.authorization as string, req.body)
 // if(isTest())
  addMatchroomToDB(req.body)
  res.send("Match added successfully")
})

export default legacyPostRouter
