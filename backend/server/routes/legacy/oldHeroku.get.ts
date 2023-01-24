import { Router, type Request, type Response } from 'express'

import { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput, checkMatchHistoryInput } from '../../middleware/checkInput'
import { getCustomHistory, getFaceitElo, getLastFive } from '../../middleware/elo'
import { addPlayerToDB } from '../../../util/database/addToDatabase'
import { InsertType } from '../../../@types/insertTypes'

const legacyGetRouter: Router = Router()

legacyGetRouter.get('/elo', checkEloInput, getFaceitElo, (req: Request, res: Response) => {
  addPlayerToDB(req.player, InsertType.ELO)
  res.send(req.player.local.responseString)
})

legacyGetRouter.get('/elobyid', checkEloByFaceitIdInput, getFaceitElo, (req: Request, res: Response) => {
  addPlayerToDB(req.player, InsertType.ELOBYID)
  res.send(req.player.local.responseString)
})

legacyGetRouter.get('/elobysteamid', checkEloBySteamIdInput, getFaceitElo, (req: Request, res: Response) => {
  addPlayerToDB(req.player, InsertType.ELOBYSTEAMID)
  res.send(req.player.local.responseString)
})

legacyGetRouter.get('/checkelo', checkEloInput, getFaceitElo, getLastFive, (req: Request, res: Response) => {
  addPlayerToDB(req.player, InsertType.CHECKELO)
  res.send(req.player.local.responseString)
})

legacyGetRouter.get('/matchhistory', checkMatchHistoryInput, getFaceitElo, getCustomHistory, (req: Request, res: Response) => {
  addPlayerToDB(req.player, InsertType.MATCHHISTORY)
  res.send(req.player.local.responseString)
})

export default legacyGetRouter
