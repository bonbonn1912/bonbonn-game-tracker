import jest, {describe, expect, test, afterAll} from '@jest/globals';
import request from 'supertest';
import server from '../backend/server/server';
import { addBody } from '../examples/webhooks/addGame';
import { delBody } from '../examples/webhooks/delGame';
import { dropDB } from '../backend/util/database/mongo';

server.listen(3000, () =>{})
describe('Get Endpoints player', () => {
    test('Get Elo by Username', async () => {
        const result = await request(server).get("/elo?username=BonBonn-_-");
        expect(result.text).toContain("BonBonn-_- is level 10")
        
    });
    test('Get Elo by Steam 64 id', async () => {
        const result = await request(server).get("/elobysteamid?id=76561198124621741");
        expect(result.text).toContain("BonBonn-_- is level 10")
        
    });
    test('Get Elo by Faceit ID', async () => {
        const result = await request(server).get("/elobyid?id=4967963b-566f-4e8a-908b-02c8315d12f1");
        expect(result.text).toContain("is level 10 with")
        
    });
    test('Get Elo (Usernot found)', async () => {
        const result = await request(server).get("/elo?username=ABCDEFG");
        expect(result.text).toBe("Invalid username")
        
    });
    test('Get Elo Invalid Steam 64 id', async () => {
        const result = await request(server).get("/elobysteamid?id=invalidsteamid");
        expect(result.text).toBe("Steam 64 Ids are made of 17 numbers (0-9)")
        
    });
    test('Get Elo Invalid Faceit ID', async () => {
        const result = await request(server).get("/elobyid?id=4967963b-INVA-LIDID-908b-02c8315d12f1");
        expect(result.text).toBe("Faceit Ids must match following pattern : ^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$")
        
    });
    test('Check elo', async () => {
        const result = await request(server).get("/matchhistory?username=BonBonn-_-&limit=150");
        expect(result.statusCode).toBe(200)
        expect(result.text).toBe("Incorrect faceit username or invalid limit (4-100)")
        
    });
    test('Check elo valid', async () => {
        const result = await request(server).get("/matchhistory?username=BonBonn-_-&limit=90");
        expect(result.status).toBe(200)
        expect(result.text).toContain("Winrate")
        
    });
    test('Check elo invalid history', async () => {
        const result = await request(server).get("/matchhistory?username=xxaliinaxx&limit=50");
        expect(result.status).toBe(200)
        expect(result.text).toContain("Play at least 3 Games to see your history")
        
    });
    afterAll(() => {
        dropDB()
    });
    
  });
describe('Post Endpoints Add Delete and Get Match', () =>{
    test('Invalid Authorization Header', async () =>{
        const result = await request(server).post("/match").set("Authorization", "4967963b-XYZZ-4e8a-fdfdfd908b-02c8315d12f1")
        expect(result.statusCode).toBe(200)
        expect(result.text).toBe("Invalid Authorization Header")
    })  
    test('Add Game', async () =>{
        const result = await request(server).post("/match")
        .set("Authorization", "4967963b-566f-4e8a-908b-02c8315d12f1")
        .send(addBody)
        expect(result.text).toBe("Match added successfully")
        expect(result.statusCode).toBe(200)
        
    })
    test('Add Second Game', async () =>{
        const result = await request(server).post("/match")
        .set("Authorization", "4967963b-566f-4e8a-908b-02c8315d12f1")
        .send(addBody)
        expect(result.text).toBe("There is already a running game for streamer 4967963b-566f-4e8a-908b-02c8315d12f1")
        expect(result.statusCode).toBe(404)
        
    })
    test('Get Match', async () => {
        const result = await request(server).get("/getmatch?key=Invalid Key");
        expect(result.text).toContain("Invalid Key")
        
    });
    test('Get Match', async () => {
        const result = await request(server).get("/getmatch?key=4967963b-566f-4e8a-908b-02c8315d12f1");
        expect(result.text).toContain("Link zum Matchroom")
        
    });
    test('Delete Game', async () =>{
        const result = await request(server).post("/match")
        .set("Authorization", "4967963b-566f-4e8a-908b-02c8315d12f1")
        .send(delBody)
        expect(result.text).toBe("Close match")
        expect(result.statusCode).toBe(200)
        
    })
    test('Get Match', async () => {
        const result = await request(server).get("/getmatch?key=4967963b-566f-4e8a-908b-02c8315d12f1");
        expect(result.text).toContain("This User is currently not")
        
    });
    afterAll(() => {
        dropDB()
    });
    
})

