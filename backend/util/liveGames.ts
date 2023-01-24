import webHookBody from "../@types/webhook";

let liveGames = new Map<string, webHookBody>
let redirectUrl: Map<string, string> = new Map([
    ["4967963b-566f-4e8a-908b-02c8315d12f1","http://edox.faceitlobby.com"],
    ["afce1ec2-1b60-47bd-88c1-bfadce96876a", "http://danielpello.faceitlobby.com"]
]
   
);

const isStreamer = (key: string): boolean =>{
    return redirectUrl.has(key)
}

const addGame = (key: string, game: webHookBody): void =>{
    liveGames.set(key, game)
}

const removeGame = (key: string) => {
    liveGames.delete(key)
}

const getGame = (key: string) : webHookBody | undefined=> {
    if(liveGames.has(key)){
        return liveGames.get(key)
    }
    return undefined
}
 
export { addGame, removeGame, getGame, isStreamer, redirectUrl }