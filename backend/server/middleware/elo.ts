import { Request, Response, NextFunction} from "express"

import faceitPlayerReponse, { faceitMatchHistory} from "../../@types/player"
import faceitElo from "../../@types/level"
import { getFaceitPlayer } from "../../util/faceit/getEloFromFaceitApi"
import { getLastFiveGames } from "../../util/faceit/getLastFiveGames"
import { faceitRankInformation } from "../../util/faceit/getEloInformation"
import { eloReponse, appendHistory, appendWinRate } from "../../util/faceit/createResponseString"
import { getCustomFaceitHistory } from "../../util/faceit/getCustomHistory"
import { faction1, faction2, matchHistory } from "../../@types/customHistory"
import { isWinnerFaction } from "../../util/faceit/isWinnerFaction"

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
        let lastFiveGames: string[] = await getLastFiveGames(req.player.player_id as string)
        req.player.local.responseString += appendHistory(lastFiveGames, ". Last 5 Games: ")
        req.player.matchHistory = lastFiveGames
    }catch{
        req.player.matchHistory = [""]
    }
    next()
}

const getCustomHistory = async (req: Request, res: Response, next: NextFunction) =>{
    try{
        let customHistory: matchHistory = await getCustomFaceitHistory(req.player.player_id, req.query.limit as any)
        let matchHistory: string[] = [];
        customHistory.items.forEach(singleGame =>{
            let winnerFaction = singleGame.results.winner == "faction1" ? singleGame.teams.faction1 : singleGame.teams.faction2;
         
            if(isWinnerFaction(req.player.player_id,winnerFaction)){
                matchHistory.push("1")
            }else{
                matchHistory.push("0")
            }
        })
        if(matchHistory.length > 3){
            req.player.matchHistory = matchHistory.reverse()
            req.player.local.responseString = appendHistory(matchHistory, `${req.player.nickname}'s last ${req.query.limit} Games: `) + appendWinRate(matchHistory)
            next()
        }else{
            res.status(404).send("Play at least 3 Games to see your history")
        }
       
     
    }catch{
        console.log("hier")
        res.status(404).send("Something went wrong")
    }

}


export { getFaceitElo, getLastFive, getCustomHistory }
   
