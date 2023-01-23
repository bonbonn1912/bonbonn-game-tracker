import { Request, Response, NextFunction} from "express"

import faceitPlayerReponse, { faceitMatchHistory} from "../../@types/player"
import faceitElo from "../../@types/level"
import { getFaceitPlayer } from "../../util/faceit/getEloFromFaceitApi"
import { getLastFiveGames } from "../../util/faceit/getLastFiveGames"
import { faceitRankInformation } from "../../util/faceit/getEloInformation"
import { eloReponse, appendHistory } from "../../util/faceit/createResponseString"

const getFaceitElo = async (req: Request, res: Response, next: NextFunction) =>{
    
    try{
        let player: faceitPlayerReponse = await getFaceitPlayer(req.query.username as string, req.query.id as string)
        let rankInformation: faceitElo = faceitRankInformation(player)
        rankInformation.responseString = eloReponse(rankInformation)
        player.local = rankInformation
        req.player = player
        next()
    }catch{
        res.status(404).send("Invalid username / steam64 id / faceit id")
    }   
}

const getLastFive = async (req: Request, res: Response, next: NextFunction) =>{
    try{
        let lastFiveGames = await getLastFiveGames(req.player.player_id as string)
        req.player.local.responseString += appendHistory(lastFiveGames)
        req.player.matchHistory = lastFiveGames
    }catch{
        req.player.matchHistory = [""]
    }
    next()
}


export { getFaceitElo, getLastFive }
   
