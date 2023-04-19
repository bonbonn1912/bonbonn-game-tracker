import { Router, type Request, type Response } from 'express'
import { addMatchroomToDB } from '../../../util/database/addToDatabase'
import { addEloInformation, validateAuthorizationHeader, validateEvent, validateStreamerGame, validateGameMode } from '../../middleware/match'
import { addGame, getGame } from '../../../util/liveGames'

const legacyPostRouter: Router = Router()

legacyPostRouter.post('/match',validateAuthorizationHeader,validateGameMode,validateEvent, validateStreamerGame, addEloInformation, (req: Request, res: Response) => {
  addGame(req.headers.authorization as string, req.body)
  addMatchroomToDB(req.body)
  res.json(getGame(req.headers.authorization as string))
})

export default legacyPostRouter
