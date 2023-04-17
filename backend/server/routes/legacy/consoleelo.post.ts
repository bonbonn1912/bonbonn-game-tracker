import { Router, type Request, type Response } from 'express'

import { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput, checkMatchHistoryInput } from '../../middleware/checkInput'
import { getCustomHistory, getFaceitElo, getLastFive } from '../../middleware/elo'
import { addPlayerToDB } from '../../../util/database/addToDatabase'
import { InsertType } from '../../../@types/insertTypes'
import { redirectToMatchroom, validateKeyInput } from '../../middleware/match'
import webHookBody from '../../../@types/webhook'
import { getGame } from '../../../util/liveGames'
import { matchRoomResponse } from '../../../util/createResponseString'
import { getFaceitPlayer } from '../../../util/faceit/player/getEloFromFaceitApi'

const consolePostRouter: Router = Router()

consolePostRouter.post("/internal/api/getMultiplePlayers",async (req: any, res: Response)=>{
    let newPlayers : any = [];
    let players = req.body.players;
  
    await Promise.all(players.map(async (player : any) => {
        let eloResp = await getFaceitPlayer("dwqd",player.steam64ID);
        player.elo = eloResp.games.csgo.faceit_elo;
        newPlayers.push(player);
       })); 
    console.log(newPlayers)
    
    res.send(newPlayers)
})
export default consolePostRouter