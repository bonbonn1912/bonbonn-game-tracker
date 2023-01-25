import { Router, type Request, type Response } from 'express'

import { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput, checkMatchHistoryInput } from '../../middleware/checkInput'
import { getCustomHistory, getFaceitElo, getLastFive } from '../../middleware/elo'
import { addPlayerToDB } from '../../../util/database/addToDatabase'
import { InsertType } from '../../../@types/insertTypes'
import { redirectToMatchroom, validateKeyInput } from '../../middleware/match'
import webHookBody from '../../../@types/webhook'
import { getGame } from '../../../util/liveGames'
import { matchRoomResponse } from '../../../util/createResponseString'

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

legacyGetRouter.get("/getmatch", validateKeyInput, (req: Request, res: Response) =>{
  let currentGame: webHookBody | undefined = getGame(req.query.key as string)
  if(currentGame == undefined){
    res.send("This User is currently not in a matchroom")
  }else{
    let reponseString: string = matchRoomResponse(currentGame, req.query.key as string)
    res.send(reponseString)
  }
})

legacyGetRouter.get("/redirect", validateKeyInput,redirectToMatchroom)

export default legacyGetRouter
