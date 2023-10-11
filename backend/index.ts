import server from './server/server'
import { SECRETS } from './config/env'
import { getLiveGames } from './util/database/addToDatabase'
import { initLiveGames } from './util/liveGames'
import webHookBody from './@types/webhook'

console.log(SECRETS.var1)
getLiveGames().then(games =>{
    console.log(games.length)
    initLiveGames(games)
})
server.listen(SECRETS.PORT, () => { console.log(`Server in container listening on port ${SECRETS.PORT}`) })
