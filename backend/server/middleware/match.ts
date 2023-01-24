import { Request, Response, NextFunction } from "express"
import { getMap } from "../../util/faceit/matchroom/getMap"
import teamFromMatch, { rosterPLayer} from "../../@types/roster"
import { getFaceitPlayer } from "../../util/faceit/player/getEloFromFaceitApi"
import { matchup, firstTeam, secondTeam } from "../../@types/webhook"
import { SECRETS } from "../../config/env"


const validateEvent = async (req: Request, res: Response, next: NextFunction) =>{
    if(req.body.event === 'match_status_configuring'){
        req.body.map = await getMap(req.body.payload.id)
        next()
    }else{
        res.send("Invalid Request oder match is closed")
    }
}

const validateAuthorizationHeader = (req: Request, res: Response, next: NextFunction) =>{
    const regexp = new RegExp(SECRETS.regex.faceitId)
    console.log(regexp.test(req.headers.authorization as string))
    if(regexp.test(req.headers.authorization as string)){
        req.body.streamer = req.headers.authorization
        next()
    }else{
        res.send("Invalid Authorization Header")
    }
    
}

const addEloInformation = async (req: Request, res: Response, next: NextFunction) =>{
    let team1: teamFromMatch = req.body.payload.teams[0]
    let team2: teamFromMatch = req.body.payload.teams[1]
    var avgTeam1: number = await getAvgElo(team1.roster)
    var avgTeam2: number = await getAvgElo(team2.roster)
    let firstTeam : firstTeam = {
        name: req.body.payload.teams[0].name,
        avgElo: avgTeam1
    }
    let secondTeam: secondTeam = {
        name: req.body.payload.teams[1].name,
        avgElo: avgTeam2
    }
    let matchup: matchup = {
        team1: firstTeam,
        team2: secondTeam
    }
    req.body.matchup = matchup
    next();
}

const getAvgElo =  async (roster: rosterPLayer[]): Promise<number> =>{
   const teamElo: number[] =  []
   let avgElo: number = 0;
   await Promise.all([
        getFaceitPlayer(roster[0].nickname),
        getFaceitPlayer(roster[1].nickname),
        getFaceitPlayer(roster[2].nickname),
        getFaceitPlayer(roster[3].nickname),
        getFaceitPlayer(roster[4].nickname)
    ]).then(players =>{
        players.forEach(singleplayer =>{
            teamElo.push(singleplayer.games.csgo.faceit_elo)
        })
        avgElo = Math.floor(teamElo.reduce((curr: number, next: number) => curr + next,0)/5)
    }).catch(err =>{
        console.log("Could not retreive avg Elo")
    })
    return avgElo
}

const validateKeyInput = (req: Request, res: Response, next: NextFunction) =>{
        const regexp = new RegExp(SECRETS.regex.faceitId)
        if(regexp.test(req.query.key as string)){
            next()
        }else{
            res.send("Invalid Key")
        }

}

export { validateEvent, addEloInformation, validateAuthorizationHeader, validateKeyInput }