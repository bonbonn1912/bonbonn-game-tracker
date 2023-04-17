import { Router, type Request, type Response } from 'express'

import { getFaceitPlayer } from '../../../util/faceit/player/getEloFromFaceitApi'

const consolePostRouter: Router = Router()

consolePostRouter.post("/internal/api/getMultiplePlayers",async (req: any, res: Response)=>{
    let newPlayers : any = [];
    let players = req.body.players;
    
    await Promise.all(players.map(async (player : any) => {
        let eloResp = await getFaceitPlayer(player.nickname,player.steam64ID);
        player.elo = eloResp.games.csgo.faceit_elo;
        newPlayers.push(player);
       })); 
    
    res.send(newPlayers)
})


export default consolePostRouter