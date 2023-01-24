import server from './server/server'
import { SECRETS } from './config/env'

console.log(SECRETS.var1)
server.listen(SECRETS.PORT, () => { console.log(`Server listening on port ${SECRETS.PORT}`) })
