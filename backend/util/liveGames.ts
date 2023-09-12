import webHookBody from "../@types/webhook";
import log from "./logging/print";

let liveGames = new Map<string, webHookBody>
let redirectUrl: Map<string, string> = new Map([
    ["4967963b-566f-4e8a-908b-02c8315d12f1","http://edox.faceitlobby.com"],
    ["afce1ec2-1b60-47bd-88c1-bfadce96876a", "http://danielpello.faceitlobby.com"],
    ["e3cefc97-36b1-4615-91d6-cdcca13e4e58","http://megalopolik.faceitlobby.com"]
]
   
);

const isStreamer = (key: string): boolean =>{
    return redirectUrl.has(key)
}

const isLive = (key: string): boolean =>{
    return liveGames.has(key)
}

const addGame = (key: string, game: webHookBody): void =>{
    liveGames.set(key, game)
    log("Added Game for Player " + key)
}

const initLiveGames = (games: webHookBody[]): void =>{
    games.forEach(game =>{
        addGame(game.streamer, game)
    })
}

const removeGame = (key: string) : void => {
   if(liveGames.has(key)){
    liveGames.delete(key)
    log("Deleted Game for Player: " + key)
   }else{
    log("Could not delete Game")
   }  
}

const getGame = (key: string) : webHookBody | undefined=> {
    if(liveGames.has(key)){
        return liveGames.get(key)
    }
    return undefined
}

const getLive = () =>{
    return liveGames;
}
 
export { addGame, removeGame, getGame, isStreamer, initLiveGames,isLive, redirectUrl, getLive }