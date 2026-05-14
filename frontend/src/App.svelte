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
	let authLoading = false;
	let oauthAvailable = true;
	let error = '';
	let result = null;
	let avatarSpec = null;
	let avatarPreviewModulePromise = null;

	$: avatarSpec = result ? buildAvatarSpec(result) : null;
	$: avatarPreviewModulePromise = result ? import('./lib/AvatarPreview.svelte') : null;

	onMount(() => {
		void loadHealth();

		const currentURL = new URL(window.location.href);
		const githubLogin = currentURL.searchParams.get('github_login');
		const authError = currentURL.searchParams.get('auth_error');

		if (githubLogin) {
			username = githubLogin;
			void analyzeForUsername(githubLogin);
		}

		if (authError) {
			error = authError;
		}

		if (githubLogin || authError) {
			currentURL.searchParams.delete('github_login');
			currentURL.searchParams.delete('auth_error');
			window.history.replaceState({}, '', currentURL);
		}
	});

	async function loadHealth() {
		try {
			const response = await fetch(`${apiBaseUrl}/api/health`);
			if (!response.ok) {
				return;
			}

			const data = await response.json();
			oauthAvailable = data.githubOAuthConfigured !== false;
		} catch {
			oauthAvailable = true;
		}
	}

	async function analyze() {
		return analyzeForUsername(username);
	}

	async function analyzeForUsername(value) {
		error = '';
		result = null;

		const normalized = value.trim();
		if (!normalized) {
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
				body: JSON.stringify({ username: normalized })
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

	function loginWithGitHub() {
		if (!oauthAvailable) {
			error = 'GitHubログインはまだ設定されていません。ユーザー名検索はそのまま使えます。';
			return;
		}

		authLoading = true;
		window.location.href = `${apiBaseUrl}/api/v1/auth/github/login`;
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

	function buildAvatarSpec(data) {
		const paletteByActivity = {
			commits: {
				primary: '#0f766e',
				secondary: '#f4bd78',
				accent: '#125b8a'
			},
			pull_requests: {
				primary: '#c77831',
				secondary: '#ffe1b6',
				accent: '#0f766e'
			},
			issues: {
				primary: '#7c5c3f',
				secondary: '#e7d2b9',
				accent: '#7a2e4d'
			},
			reviews: {
				primary: '#375a7f',
				secondary: '#dce7f5',
				accent: '#0f766e'
			}
		};

		const palette = paletteByActivity[data.stats.dominantActivity] ?? paletteByActivity.commits;
		const total = data.stats.totalContributions;
		const activeDays = data.stats.activeDays;
		const streak = data.stats.longestStreak;
		const peak = data.stats.peakDayContributions;
		const weekendBias = data.stats.weekendRatio;
		const repoCount = data.topRepositories.length;

		return {
			palette,
			torsoHeight: clamp(1.6 + activeDays / 260, 1.7, 2.45),
			torsoRadius: clamp(0.5 + total / 900, 0.54, 0.9),
			headRadius: clamp(0.72 + peak / 40, 0.75, 1.1),
			limbRadius: clamp(0.11 + data.stats.pullRequestCount / 250, 0.12, 0.22),
			limbLength: clamp(0.95 + total / 1000, 1.02, 1.45),
			legRadius: clamp(0.14 + data.stats.commitCount / 420, 0.15, 0.24),
			legLength: clamp(1.25 + activeDays / 300, 1.25, 1.75),
			finCount: clamp(2 + Math.floor(total / 180), 2, 7),
			finHeight: clamp(0.7 + peak / 18, 0.75, 1.35),
			antennaCount: clamp(1 + Math.floor(streak / 18), 1, 4),
			antennaHeight: clamp(0.55 + streak / 80, 0.6, 1.1),
			satelliteCount: clamp(repoCount, 1, 5),
			badgeCount: clamp(
				Math.max(data.stats.pullRequestCount, data.stats.reviewCount, data.stats.issueCount) > 0
					? 1 + Math.floor((data.stats.pullRequestCount + data.stats.reviewCount) / 18)
					: 1,
				1,
				4
			),
			weekendBias
		};
	}

	function getAvatarParameters(spec, stats) {
		return [
			{
				part: '胴体',
				value: `${spec.torsoHeight.toFixed(2)}`,
				source: `活動日数 ${formatNumber(stats.activeDays)}日`,
				meaning: '年間を通した継続力'
			},
			{
				part: '頭部',
				value: `${spec.headRadius.toFixed(2)}`,
				source: `最多活動日 ${formatNumber(stats.peakDayContributions)}`,
				meaning: '1日に集中して出せる瞬発力'
			},
			{
				part: '背面フィン',
				value: `${spec.finCount}枚`,
				source: `総コントリビューション ${formatNumber(stats.totalContributions)}`,
				meaning: '積み上げたアウトプット量'
			},
			{
				part: 'アンテナ',
				value: `${spec.antennaCount}本`,
				source: `最長連続日数 ${formatNumber(stats.longestStreak)}日`,
				meaning: '連続して走り続ける持久力'
			},
			{
				part: 'サテライト',
				value: `${spec.satelliteCount}個`,
				source: `主要リポジトリ ${stats.topRepositoryCount}件`,
				meaning: '活動の広がりと探索範囲'
			},
			{
				part: '配色',
				value: formatActivityLabel(stats.dominantActivity),
				source: `主要活動 ${formatActivityLabel(stats.dominantActivity)}`,
				meaning: 'どの行動がいちばん強く出ているか'
			},
			{
				part: '胸バッジ',
				value: `${spec.badgeCount}個`,
				source: `PR+レビュー ${formatNumber(stats.pullRequestCount + stats.reviewCount)}`,
				meaning: '共同開発への関与度'
			}
		];
	}

	function clamp(value, min, max) {
		return Math.min(Math.max(value, min), max);
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
	</section>

	<section class="panel form-panel">
		<div class="form-header">
			<div>
				<p class="eyebrow">分析入力</p>
				<h2>GitHubユーザー名</h2>
			</div>
			<p class="form-note">
				ユーザー名で直接検索するか、GitHub でログインして自分のアカウントをすぐ分析できます。
			</p>
		</div>

		<div class="oauth-row">
			<button class="oauth-button" on:click={loginWithGitHub} disabled={loading || authLoading || !oauthAvailable}>
				{#if authLoading}
					GitHubへ移動中...
				{:else}
					GitHubでログインして分析
				{/if}
			</button>
			<p class="oauth-note">ログイン後は GitHub のユーザー名を自動入力して、そのまま分析を開始します。</p>
		</div>

		<div class="input-row">
			<input
				id="username"
				bind:value={username}
				type="text"
				placeholder="octocat"
				autocomplete="off"
				on:keydown={(event) => event.key === 'Enter' && analyzeForUsername(username)}
			/>
			<button on:click={() => analyzeForUsername(username)} disabled={loading || authLoading}>
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

		<section class="result-card model-section">
			<div class="model-copy">
				<p class="eyebrow">コード生成3Dモデル</p>
				<h2>分析結果から組み上がるアバター</h2>
				<p class="body-copy">
					このモデルは AI 画像生成ではなく、活動量・streak・主要活動・リポジトリ数から形状と色を
					決めてコードで組み立てています。
				</p>

				<div class="model-parameters">
					{#each getAvatarParameters(avatarSpec, { ...result.stats, topRepositoryCount: result.topRepositories.length }) as parameter}
						<div class="model-parameter-card">
							<div class="model-parameter-head">
								<strong>{parameter.part}</strong>
								<span>{parameter.value}</span>
							</div>
							<p>{parameter.meaning}</p>
							<small>{parameter.source}</small>
						</div>
					{/each}
				</div>
			</div>

			<div class="model-stage">
				{#if avatarPreviewModulePromise}
					{#await avatarPreviewModulePromise}
						<div class="model-loading">3Dプレビューを準備しています...</div>
					{:then module}
						<svelte:component this={module.default} spec={avatarSpec} />
					{:catch}
						<div class="model-loading">3Dプレビューの読み込みに失敗しました。</div>
					{/await}
				{/if}
			</div>
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
