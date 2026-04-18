<script>
	import { onMount } from 'svelte';

	const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? '';

	const statLabels = {
		totalContributions: '総コントリビューション',
		activeDays: '活動日数',
		longestStreak: '最長連続日数',
		currentStreak: '現在の連続日数',
		peakDayContributions: '最多活動日',
		weekendRatio: '週末比率'
	};

	let username = '';
	let loading = false;
	let error = '';
	let result = null;
	let health = null;
	let healthLoading = true;

	onMount(() => {
		loadHealth();
	});

	async function loadHealth() {
		healthLoading = true;

		try {
			const response = await fetch(`${apiBaseUrl}/api/health`);
			if (!response.ok) {
				throw new Error('health check failed');
			}

			health = await response.json();
		} catch {
			health = {
				status: 'down',
				githubTokenConfigured: false
			};
		} finally {
			healthLoading = false;
		}
	}

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
				body: JSON.stringify({ username: username.trim() })
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

	function formatPercent(value) {
		return `${Math.round(value * 100)}%`;
	}

	function formatActivityLabel(value) {
		switch (value) {
			case 'pull_requests':
				return 'プルリクエスト';
			case 'issues':
				return 'Issue';
			case 'reviews':
				return 'レビュー';
			default:
				return 'コミット';
		}
	}

	function formatNumber(value) {
		return new Intl.NumberFormat('ja-JP').format(value);
	}

	function getPrimaryStats(stats) {
		return [
			{
				key: 'totalContributions',
				label: statLabels.totalContributions,
				value: formatNumber(stats.totalContributions)
			},
			{
				key: 'activeDays',
				label: statLabels.activeDays,
				value: `${formatNumber(stats.activeDays)}日`
			},
			{
				key: 'longestStreak',
				label: statLabels.longestStreak,
				value: `${formatNumber(stats.longestStreak)}日`
			},
			{
				key: 'currentStreak',
				label: statLabels.currentStreak,
				value: `${formatNumber(stats.currentStreak)}日`
			},
			{
				key: 'peakDayContributions',
				label: statLabels.peakDayContributions,
				value: formatNumber(stats.peakDayContributions)
			},
			{
				key: 'weekendRatio',
				label: statLabels.weekendRatio,
				value: formatPercent(stats.weekendRatio)
			}
		];
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
		<div class="hero-copy">
			<p class="eyebrow">Contribution Radar</p>
			<h1>GitHub の活動ログを<br />ペルソナと傾向に変換する</h1>
			<p class="lead">
				GitHub GraphQL API から過去1年の活動を読み取り、streak、週末比率、主要リポジトリ、
				主なアクティビティ種別までまとめて可視化します。
			</p>
		</div>

		<div class="status-panel">
			<p class="eyebrow">バックエンド状態</p>
			{#if healthLoading}
				<p class="status-line">API 状態を確認しています...</p>
			{:else}
				<div class="status-list">
					<div class="status-item">
						<span>API</span>
						<strong class:ok={health?.status === 'ok'} class:bad={health?.status !== 'ok'}>
							{health?.status === 'ok' ? '稼働中' : '停止中'}
						</strong>
					</div>
					<div class="status-item">
						<span>GitHubトークン</span>
						<strong class:ok={health?.githubTokenConfigured} class:bad={!health?.githubTokenConfigured}>
							{health?.githubTokenConfigured ? '設定済み' : '未設定'}
						</strong>
					</div>
				</div>
			{/if}
		</div>
	</section>

	<section class="panel form-panel">
		<div class="form-header">
			<div>
				<p class="eyebrow">分析入力</p>
				<h2>GitHubユーザー名</h2>
			</div>
			<p class="form-note">`GITHUB_TOKEN` が設定されていると、実データ分析を返します。</p>
		</div>

		<div class="input-row">
			<input
				id="username"
				bind:value={username}
				type="text"
				placeholder="octocat"
				autocomplete="off"
				on:keydown={(event) => event.key === 'Enter' && analyze()}
			/>
			<button on:click={analyze} disabled={loading}>
				{#if loading}
					分析中...
				{:else}
					分析する
				{/if}
			</button>
		</div>

		{#if error}
			<p class="message error">{error}</p>
		{/if}
	</section>

	{#if result}
		<section class="result-card hero-result">
			<div class="profile-block">
				{#if result.profile.avatarUrl}
					<img class="avatar" src={result.profile.avatarUrl} alt={result.username} />
				{/if}

				<div class="profile-copy">
					<p class="eyebrow">Persona</p>
					<h2>{result.personaTitle}</h2>
					<p class="summary">{result.summary}</p>

					<div class="profile-meta">
						<span>@{result.username}</span>
						{#if result.profile.name}
							<span>{result.profile.name}</span>
						{/if}
						{#if result.profile.url}
							<a href={result.profile.url} target="_blank" rel="noreferrer">GitHub Profile</a>
						{/if}
					</div>
				</div>
			</div>

			<div class="pill-row">
				<div class="pill">
					<span>おすすめアセット</span>
					<strong>{result.recommendedAsset}</strong>
				</div>
				<div class="pill">
					<span>主要アクティビティ</span>
					<strong>{formatActivityLabel(result.stats.dominantActivity)}</strong>
				</div>
				<div class="pill">
					<span>分析期間</span>
					<strong>{result.stats.from} - {result.stats.to}</strong>
				</div>
			</div>
		</section>

		<section class="stats-grid">
			{#each getPrimaryStats(result.stats) as stat}
				<article class="stat-card">
					<p>{stat.label}</p>
					<h3>{stat.value}</h3>
				</article>
			{/each}
		</section>

		<section class="detail-grid">
			<article class="result-card">
				<p class="eyebrow">特徴</p>
				<ul class="trait-list">
					{#each result.traits as trait}
						<li>{trait}</li>
					{/each}
				</ul>
			</article>

			<article class="result-card">
				<p class="eyebrow">ビジュアル方針</p>
				<p class="body-copy">{result.visualDirection}</p>
				<div class="signal-block">
					<h3>活動シグナル</h3>
					<p>{result.contributionSignal}</p>
				</div>
			</article>

			<article class="result-card">
				<p class="eyebrow">活動内訳</p>
				<div class="activity-list">
					<div>
						<span>コミット</span>
						<strong>{formatNumber(result.stats.commitCount)}</strong>
					</div>
					<div>
						<span>プルリクエスト</span>
						<strong>{formatNumber(result.stats.pullRequestCount)}</strong>
					</div>
					<div>
						<span>Issue</span>
						<strong>{formatNumber(result.stats.issueCount)}</strong>
					</div>
					<div>
						<span>レビュー</span>
						<strong>{formatNumber(result.stats.reviewCount)}</strong>
					</div>
					<div>
						<span>最も動いた曜日</span>
						<strong>{result.stats.busiestWeekday}</strong>
					</div>
				</div>
			</article>
		</section>

		<section class="result-card repository-section">
			<div class="section-header">
				<div>
					<p class="eyebrow">主要リポジトリ</p>
					<h2>主な活動先</h2>
				</div>
			</div>

			{#if result.topRepositories.length > 0}
				<div class="repository-list">
					{#each result.topRepositories as repository}
						<a class="repository-card" href={repository.url} target="_blank" rel="noreferrer">
							<div class="repository-head">
								<h3>{repository.nameWithOwner}</h3>
								<span>合計 {formatNumber(repository.total)}</span>
							</div>
							<div class="repository-stats">
								<span>コミット {formatNumber(repository.commits)}</span>
								<span>PR {formatNumber(repository.pullRequests)}</span>
								<span>Issue {formatNumber(repository.issues)}</span>
								<span>レビュー {formatNumber(repository.reviews)}</span>
							</div>
						</a>
					{/each}
				</div>
			{:else}
				<p class="body-copy">リポジトリ単位の活動はまだ取得できていません。</p>
			{/if}
		</section>
	{/if}
</main>
