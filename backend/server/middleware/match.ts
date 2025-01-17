import { Request, Response, NextFunction } from "express"
import { getMap } from "../../util/faceit/matchroom/getMap"
import teamFromMatch, { rosterPLayer} from "../../@types/roster"
import { getFaceitPlayer } from "../../util/faceit/player/getEloFromFaceitApi"
import webHookBody, { matchup, firstTeam, secondTeam } from "../../@types/webhook"
import { SECRETS } from "../../env"
import { removeGame, getGame, isLive, redirectUrl } from "../../util/liveGames"
import { alreadyLive, logInsert, mongoUpdate } from "../../util/database/addToDatabase"



const validateEvent = async (req: Request, res: Response, next: NextFunction) =>{
    if(req.body.event === 'match_status_configuring'){
        req.body.map = await getMap(req.body.payload.id)
        next()
    }else if(req.body.event !== 'match_object_created'){
        removeGame(req.headers.authorization as string)
        mongoUpdate(req.headers.authorization as string);
        res.send("Close match")
    }else{
        res.send("Invalid Request oder match is already closed")
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
   console.log(`Fetching Elo for: ${roster.map(player => player.nickname)}` )
   await Promise.all([
        getFaceitPlayer(roster[0].nickname),
        getFaceitPlayer(roster[1].nickname),
        getFaceitPlayer(roster[2].nickname),
        getFaceitPlayer(roster[3].nickname),
        getFaceitPlayer(roster[4].nickname)
    ]).then(players =>{
        players.forEach(singleplayer =>{
            teamElo.push(singleplayer.games.cs2?.faceit_elo as number)
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

const validateStreamerGame = async(req: Request, res: Response, next: NextFunction) =>{
    const live = await alreadyLive(req.headers.authorization as string)
    if(live){
        res.status(404).send("There is already a running game for streamer " + req.headers.authorization)
    }else{
        next()
    }
}

const redirectToMatchroom = async (req: Request, res: Response) =>{
    try{
       if (req.query.key !== "f335cfd1-3a92-4365-9b25-2f0e82a6052f") {
    logInsert({ 
        text: "Accessing Game for " + req.query.key + 
              " Streamer: " + redirectUrl.get(req.query.key as string) + 
              " IP: " + (req.socket.remoteAddress || req.headers['x-forwarded-for'])
    }, "logs");
}
    }catch(e){
        console.log("Could not log access")
    }
    if(isLive(req.query.key as string)){
        let game: webHookBody | undefined = getGame(req.query.key as string)
        res.redirect(`https://www.faceit.com/en/csgo/room/${game?.payload.id}`)
    }else{
        res.send(`Player with id: ${req.query.key} is currently not ingame`)
    }
}

const validateGameMode = async (req: Request, res: Response, next: NextFunction) =>{
    if(req.body.payload.entity.type === "matchmaking" || req.body.payload.entity.type  === "hub"){
        next();
    }else{
        res.send("Invalid Gamemode")
    }
}


export { validateGameMode,validateEvent, addEloInformation, validateAuthorizationHeader, validateKeyInput, validateStreamerGame, redirectToMatchroom }
