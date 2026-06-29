use spacetimedb::{Identity, ReducerContext, Table, Timestamp};

const WORLD_WIDTH: f64 = 2400.0;
const WORLD_HEIGHT: f64 = 1800.0;
const STARTING_MASS: f64 = 22.0;
const FOOD_COUNT: u64 = 260;

#[spacetimedb::table(accessor = player, public)]
#[derive(Clone)]
pub struct Player {
    #[primary_key]
    identity: Identity,
    name: String,
    x: f64,
    y: f64,
    target_x: f64,
    target_y: f64,
    radius: f64,
    mass: f64,
    color: String,
    alive: bool,
    updated_at: Timestamp,
}

#[spacetimedb::table(accessor = food, public)]
#[derive(Clone)]
pub struct Food {
    #[primary_key]
    id: u64,
    x: f64,
    y: f64,
    mass: f64,
}

#[spacetimedb::table(accessor = world_counter)]
#[derive(Clone)]
pub struct WorldCounter {
    #[primary_key]
    id: u64,
    next_food: u64,
}

#[spacetimedb::reducer(init)]
pub fn init(ctx: &ReducerContext) {
    if ctx.db.world_counter().id().find(0).is_none() {
        ctx.db.world_counter().insert(WorldCounter {
            id: 0,
            next_food: 1,
        });
    }
    seed_food(ctx);
}

#[spacetimedb::reducer(client_disconnected)]
pub fn client_disconnected(ctx: &ReducerContext) {
    let sender = ctx.sender();
    if let Some(existing) = ctx.db.player().identity().find(sender) {
        ctx.db.player().delete(existing);
    }
}

#[spacetimedb::reducer]
pub fn join_game(ctx: &ReducerContext, name: String) {
    let sender = ctx.sender();
    if let Some(existing) = ctx.db.player().identity().find(sender) {
        ctx.db.player().delete(existing);
    }
    let slot = ctx.db.player().iter().count() as u64;
    let x = 160.0 + ((slot * 311) % 2000) as f64;
    let y = 160.0 + ((slot * 197) % 1400) as f64;
    let mass = STARTING_MASS;
    ctx.db.player().insert(Player {
        identity: sender,
        name: sanitize_name(name),
        x,
        y,
        target_x: x,
        target_y: y,
        radius: radius_for_mass(mass),
        mass,
        color: color_for_slot(slot),
        alive: true,
        updated_at: ctx.timestamp,
    });
}

#[spacetimedb::reducer]
pub fn set_input(ctx: &ReducerContext, target_x: f64, target_y: f64) {
    let sender = ctx.sender();
    let Some(mut player) = ctx.db.player().identity().find(sender) else {
        return;
    };
    if !player.alive {
        return;
    }
    player.target_x = clamp(target_x, 0.0, WORLD_WIDTH);
    player.target_y = clamp(target_y, 0.0, WORLD_HEIGHT);
    step_player(&mut player);
    eat_food(ctx, &mut player);
    eat_players(ctx, &mut player);
    player.updated_at = ctx.timestamp;
    ctx.db.player().identity().update(player);
    refill_food(ctx);
}

#[spacetimedb::reducer]
pub fn respawn(ctx: &ReducerContext) {
    let sender = ctx.sender();
    let name = ctx
        .db
        .player()
        .identity()
        .find(sender)
        .map(|p| p.name)
        .unwrap_or_else(|| "pilot".to_string());
    join_game(ctx, name);
}

fn step_player(player: &mut Player) {
    let dx = player.target_x - player.x;
    let dy = player.target_y - player.y;
    let dist = (dx * dx + dy * dy).sqrt();
    if dist <= 0.1 {
        return;
    }
    let speed = (5.4 - player.radius * 0.035).max(1.4);
    player.x = clamp(player.x + dx / dist * speed, player.radius, WORLD_WIDTH - player.radius);
    player.y = clamp(player.y + dy / dist * speed, player.radius, WORLD_HEIGHT - player.radius);
}

fn eat_food(ctx: &ReducerContext, player: &mut Player) {
    let mut eaten = Vec::new();
    for pellet in ctx.db.food().iter() {
        let dist = distance(player.x, player.y, pellet.x, pellet.y);
        if dist < player.radius {
            player.mass += pellet.mass * 0.42;
            player.radius = radius_for_mass(player.mass);
            eaten.push(pellet);
        }
    }
    for pellet in eaten {
        ctx.db.food().delete(pellet);
    }
}

fn eat_players(ctx: &ReducerContext, player: &mut Player) {
    let mut victims = Vec::new();
    for other in ctx.db.player().iter() {
        if other.identity == player.identity || !other.alive {
            continue;
        }
        if player.radius <= other.radius * 1.12 {
            continue;
        }
        if distance(player.x, player.y, other.x, other.y) < player.radius - other.radius * 0.25 {
            victims.push(other);
        }
    }
    for mut victim in victims {
        player.mass += victim.mass * 0.75;
        player.radius = radius_for_mass(player.mass);
        victim.alive = false;
        ctx.db.player().identity().update(victim);
    }
}

fn seed_food(ctx: &ReducerContext) {
    while ctx.db.food().iter().count() < FOOD_COUNT as usize {
        spawn_food(ctx);
    }
}

fn refill_food(ctx: &ReducerContext) {
    while ctx.db.food().iter().count() < FOOD_COUNT as usize {
        spawn_food(ctx);
    }
}

fn spawn_food(ctx: &ReducerContext) {
    let id = next_food_id(ctx);
    ctx.db.food().insert(Food {
        id,
        x: deterministic_coord(id, 197, WORLD_WIDTH),
        y: deterministic_coord(id, 313, WORLD_HEIGHT),
        mass: 2.0 + (id % 4) as f64,
    });
}

fn next_food_id(ctx: &ReducerContext) -> u64 {
    let mut counter = ctx
        .db
        .world_counter()
        .id()
        .find(0)
        .unwrap_or(WorldCounter {
            id: 0,
            next_food: 1,
        });
    let id = counter.next_food;
    counter.next_food += 1;
    ctx.db.world_counter().id().update(counter);
    id
}

fn deterministic_coord(id: u64, multiplier: u64, max: f64) -> f64 {
    let usable = max - 64.0;
    32.0 + ((id * multiplier) % usable as u64) as f64
}

fn radius_for_mass(mass: f64) -> f64 {
    mass.sqrt() * 4.7
}

fn distance(ax: f64, ay: f64, bx: f64, by: f64) -> f64 {
    let dx = ax - bx;
    let dy = ay - by;
    (dx * dx + dy * dy).sqrt()
}

fn clamp(value: f64, min: f64, max: f64) -> f64 {
    value.max(min).min(max)
}

fn sanitize_name(name: String) -> String {
    let trimmed = name.trim();
    if trimmed.is_empty() {
        return "pilot".to_string();
    }
    trimmed.chars().take(16).collect()
}

fn color_for_slot(slot: u64) -> String {
    let colors = [
        "#6dd8c7", "#ff719a", "#ffd166", "#9d8cff", "#66a6ff", "#7be495",
    ];
    colors[(slot as usize) % colors.len()].to_string()
}
