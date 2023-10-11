import server from './server/server'
import { SECRETS } from './env'
import { getLiveGames } from './util/database/addToDatabase'
import { initLiveGames } from './util/liveGames'
import webHookBody from './@types/webhook'

console.log(SECRETS.var1)
getLiveGames().then(games =>{
    console.log(games.length)
    initLiveGames(games)
})
console.log("This has been updated")
server.listen(SECRETS.PORT, () => { console.log(`Server In Container Listening On Port: ${SECRETS.PORT}`) })
