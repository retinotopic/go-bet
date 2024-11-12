
<script lang="ts">
    interface Ratings {
        username: string ;
        rating: string ;
    }
	import { onMount } from 'svelte';
	let socket: WebSocket;
	let ConnState = $state< 'authorized' | 'unauthorized' | 'connerror'| 'loading' >('loading');
    let ratings = $state<Ratings[]>([]);
    let gameState = $state< 'find' | 'waiting' | 'ready' >('find');
    let counter = $state(0);
    let gameUrl = $state('');
	onMount(() => {
        connectWebSocket();
    });

    function sendmsg(msg :string) {
		socket.send(msg)
    };
    function connectWebSocket() {
        socket = new WebSocket('ws://localhost:8080/hub');
        ConnState = 'loading'
        socket.onerror = () => {
            ConnState = 'connerror';
        };

		socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (!data) {
                ConnState = 'connerror';
                return
            }
            
            if (data[0] && data[0].rating && data[0].username) {
                ratings = data as Ratings[];
            } 

			let num = parseInt(data.user_id, 10)
			if (!num){
				ConnState = 'unauthorized';
				return;
			}
			ConnState = 'authorized';


			if (data.url){
				gameUrl = data.url
			}

			num = parseInt(data.counter, 10)
			if (num){
				counter = num
			}


			if (gameUrl) {
				gameState = 'ready';
                counter = 0
                socket.close()
			} else if (counter === -1) {
				gameState = 'find';
			} else {
				gameState = 'waiting';
			}
            
    };
    }


</script>

{#if ConnState === 'authorized'}
	<div class="hub-container">
		<div class="game-section">
			{#if gameState === 'find'}
				<button class="find-game" onclick={()=>{sendmsg("active")}}>Find Game</button>
			{:else if gameState === 'waiting'}
				<div class="circles">
					{#each Array(8) as _, i}
						<div class="circle" class:active={i < counter}></div>
					{/each}
				</div>
                <button class="find-game" onclick={()=>{sendmsg("inactive")}}>cancel search</button>
			{:else if gameState === 'ready'}
				<a href={gameUrl} class="game-link">Join Game</a>
			{/if}
		</div>

		<div class="ratings-section">
			<h3>Ratings</h3>
			<ul>
				{#each ratings as {username, rating}}
					<li>{username} - {rating}</li>
				{/each}
			</ul>
		</div>
	</div>
{:else if ConnState === 'unauthorized'}
	<div class="connection-error">
		<p>Login with google</p>
		<a href="URL">LOGIN</a>
	</div>
{:else if ConnState === 'connerror'}
    {connectWebSocket}
{:else}
	<p>...LOADING...</p>
{/if}

<style>
    .hub-container {
        display: flex;
        gap: 2rem;
    }

    .game-section {
        flex: 1;
    }

    .ratings-section {
        width: 300px;
    }

    .circles {
        display: flex;
        gap: 0.5rem;
    }

    .circle {
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background-color: #ccc;
    }

    .circle.active {
        background-color: #4CAF50;
    }

    .find-game {
        padding: 0.5rem 1rem;
        background-color: #2196F3;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
    }

    .game-link {
        display: inline-block;
        padding: 0.5rem 1rem;
        background-color: #4CAF50;
        color: white;
        text-decoration: none;
        border-radius: 4px;
    }

    .ratings-section ul {
        list-style: none;
        padding: 0;
    }

    .ratings-section li {
        padding: 0.5rem 0;
        border-bottom: 1px solid #eee;
    }
</style>