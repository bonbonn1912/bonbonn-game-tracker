import webHookBody from "../@types/webhook";

let liveGames = new Map<string, webHookBody>
let redirectUrl: Map<string, string> = new Map([
    ["4967963b-566f-4e8a-908b-02c8315d12f1","http://edox.faceitlobby.com"],
    ["afce1ec2-1b60-47bd-88c1-bfadce96876a", "http://danielpello.faceitlobby.com"],
    ["e3cefc97-36b1-4615-91d6-cdcca13e4e58","http://megalopolik.faceitlobby.com"],
    ["f335cfd1-3a92-4365-9b25-2f0e82a6052f","http://mrc9cs.faceitlobby.com"],
     ["e64b6d73-6313-4527-b313-a852e6368cf2","http://redcs.faceitlobby.com"],
     ["c77a24f9-4126-4321-9cc5-4dfcf69b50ee","http://bulmi.faceitlobby.com"],
     ["f8bd2fd4-1860-441e-8851-3469dd3a1a55","http://tiziaN.faceitlobby.com"],
 //   ["179d5eb7-be1a-473a-8bc5-36310ebc0f28","http://missmoeppi.faceitlobby.com"]
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
    console.log("Added Game for Player " + key)
}

const initLiveGames = (games: webHookBody[]): void =>{
    games.forEach(game =>{
        addGame(game.streamer, game)
    })
}

const removeGame = (key: string) : void => {
   if(liveGames.has(key)){
    liveGames.delete(key)
    console.log("Deleted Game for Player: " + key)
   }else{
    console.log("Could not delete Game")
   }  
}

const getGame = (key: string) : webHookBody | undefined=> {
    if(liveGames.has(key)){
        return liveGames.get(key)
    }
    return undefined
}
 
export { addGame, removeGame, getGame, isStreamer, initLiveGames,isLive, redirectUrl }
