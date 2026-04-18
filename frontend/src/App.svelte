<script>
	const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? '';

	let username = '';
	let loading = false;
	let error = '';
	let result = null;

	async function analyze() {
		error = '';
		result = null;

		if (!username.trim()) {
			error = 'GitHubユーザー名を入力してください。';
			return;
		}

		loading = true;

		try {
			const response = await fetch(`${apiBaseUrl}/api/v1/analyze`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ username })
			});

			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error ?? '分析に失敗しました。');
			}

			result = data;
		} catch (err) {
			error = err instanceof Error ? err.message : '不明なエラーが発生しました。';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>GitHub Contribution Visualizer</title>
	<meta
		name="description"
		content="GitHubの草をもとに、開発スタイルをビジュアル化するプロトタイプ"
	/>
</svelte:head>

<main class="shell">
	<section class="hero">
		<p class="eyebrow">Prototype</p>
		<h1>GitHubの草から<br />開発スタイルを可視化する</h1>
		<p class="lead">
			ユーザー名を入れると、活動傾向からペルソナとビジュアル方向性を返す最小プロトタイプです。
		</p>
	</section>

	<section class="panel">
		<label for="username">GitHub Username</label>
		<div class="input-row">
			<input
				id="username"
				bind:value={username}
				type="text"
				placeholder="octocat"
				autocomplete="off"
			/>
			<button on:click={analyze} disabled={loading}>
				{#if loading}
					Analyzing...
				{:else}
					Analyze
				{/if}
			</button>
		</div>

		{#if error}
			<p class="message error">{error}</p>
		{/if}
	</section>

	{#if result}
		<section class="result-card">
			<div class="result-header">
				<p class="eyebrow">Persona</p>
				<h2>{result.personaTitle}</h2>
				<p>{result.summary}</p>
			</div>

			<div class="result-grid">
				<div>
					<h3>Traits</h3>
					<ul>
						{#each result.traits as trait}
							<li>{trait}</li>
						{/each}
					</ul>
				</div>

				<div>
					<h3>Recommended Asset</h3>
					<p>{result.recommendedAsset}</p>
					<h3>Visual Direction</h3>
					<p>{result.visualDirection}</p>
				</div>

				<div>
					<h3>Contribution Signal</h3>
					<p>{result.contributionSignal}</p>
				</div>
			</div>
		</section>
	{/if}
</main>
