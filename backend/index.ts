import server from './server/server'
import { SECRETS } from './config/env'
import { getLiveGames } from './util/database/mongo'
import { initLiveGames } from './util/liveGames'
import webHookBody from './@types/webhook'
import log from './util/logging/print'


getLiveGames().then(games =>{
    log(games.length)
    initLiveGames(games)
})
server.listen(SECRETS.PORT, () => { log(`Server listening on port ${SECRETS.PORT}`) })
