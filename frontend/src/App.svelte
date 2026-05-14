<script>
	import { onMount } from 'svelte';

	const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? '';
	const avatarPreviewModulePromise = import('./lib/AvatarPreview.svelte');

	const statLabels = {
		totalContributions: '総コントリビューション',
		activeDays: '活動日数',
		longestStreak: '最長連続日数',
		currentStreak: '現在の連続日数',
		peakDayContributions: '最多活動日',
		weekendRatio: '週末比率'
	};

	let route = 'home';
	let username = '';
	let loading = false;
	let authLoading = false;
	let oauthAvailable = true;
	let error = '';
	let currentUser = null;
	let showcaseUsers = [];
	let searchResult = null;

	$: showcaseDisplayUsers = currentUser
		? showcaseUsers.filter((user) => user.username !== currentUser.username)
		: showcaseUsers;

	onMount(() => {
		updateRoute();
		window.addEventListener('popstate', updateRoute);

		void initialize();

		return () => {
			window.removeEventListener('popstate', updateRoute);
		};
	});

	async function initialize() {
		await Promise.all([loadHealth(), loadCurrentUser(), loadShowcase()]);

		const currentURL = new URL(window.location.href);
		const authError = currentURL.searchParams.get('auth_error');
		if (authError) {
			error = authError;
			currentURL.searchParams.delete('auth_error');
			window.history.replaceState({}, '', currentURL);
		}
	}

	function updateRoute() {
		route = window.location.pathname === '/my-page' ? 'my-page' : 'home';
	}

	function navigateTo(path) {
		if (window.location.pathname === path) {
			return;
		}

		window.history.pushState({}, '', path);
		updateRoute();
		error = '';
	}

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

	async function loadCurrentUser() {
		try {
			const response = await fetch(`${apiBaseUrl}/api/v1/me`);
			if (response.status === 401) {
				currentUser = null;
				return;
			}

			const data = await response.json();
			if (!response.ok) {
				throw new Error(data.error ?? 'ログイン情報の取得に失敗しました。');
			}

			currentUser = data;
		} catch (err) {
			currentUser = null;
			error = err instanceof Error ? err.message : 'ログイン情報の取得に失敗しました。';
		}
	}

	async function loadShowcase() {
		try {
			const response = await fetch(`${apiBaseUrl}/api/v1/showcase`);
			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error ?? 'ショーケースの取得に失敗しました。');
			}

			showcaseUsers = data.users ?? [];
		} catch (err) {
			showcaseUsers = [];
			error = err instanceof Error ? err.message : 'ショーケースの取得に失敗しました。';
		}
	}

	async function analyzeForUsername(value) {
		error = '';
		searchResult = null;

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

			searchResult = data;
		} catch (err) {
			error = err instanceof Error ? err.message : '不明なエラーが発生しました。';
		} finally {
			loading = false;
		}
	}

	function loginWithGitHub() {
		if (!oauthAvailable) {
			error = 'GitHubログインはまだ設定されていません。';
			return;
		}

		authLoading = true;
		window.location.href = `${apiBaseUrl}/api/v1/auth/github/login`;
	}

	async function logout() {
		error = '';
		authLoading = true;

		try {
			const response = await fetch(`${apiBaseUrl}/api/v1/logout`, {
				method: 'POST'
			});
			const data = await response.json();

			if (!response.ok) {
				throw new Error(data.error ?? 'ログアウトに失敗しました。');
			}

			currentUser = null;
			if (route === 'my-page') {
				navigateTo('/');
			}
			await loadShowcase();
		} catch (err) {
			error = err instanceof Error ? err.message : 'ログアウトに失敗しました。';
		} finally {
			authLoading = false;
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
		content="GitHubの草をもとに、ログイン中ユーザーの3Dモデルと自分の活動傾向を可視化する"
	/>
</svelte:head>

<main class="shell">
	<header class="site-header">
		<button class="brand" on:click={() => navigateTo('/')}>
			<span class="brand-mark"></span>
			<span>
				<strong>Contribution Radar</strong>
				<small>GitHub activity avatars</small>
			</span>
		</button>

		<nav class="header-nav">
			<button class:active={route === 'home'} on:click={() => navigateTo('/')}>トップ</button>
			<button class:active={route === 'my-page'} on:click={() => navigateTo('/my-page')}>マイページ</button>
		</nav>

		<div class="header-actions">
			{#if currentUser}
				<div class="header-user">
					{#if currentUser.profile.avatarUrl}
						<img class="header-avatar" src={currentUser.profile.avatarUrl} alt={currentUser.username} />
					{/if}
					<div>
						<strong>@{currentUser.username}</strong>
						<small>{currentUser.personaTitle}</small>
					</div>
				</div>
				<button class="ghost-button" on:click={() => navigateTo('/my-page')}>自分を見る</button>
				<button class="ghost-button" on:click={logout} disabled={authLoading}>ログアウト</button>
			{:else}
				<button class="oauth-button" on:click={loginWithGitHub} disabled={authLoading || !oauthAvailable}>
					{authLoading ? 'GitHubへ移動中...' : 'GitHubでログイン'}
				</button>
			{/if}
		</div>
	</header>

	{#if error}
		<section class="panel message-panel">
			<p class="message error">{error}</p>
		</section>
	{/if}

	{#if route === 'home'}
		<section class="hero home-hero">
			<div class="hero-copy">
				<p class="eyebrow">Live Showcase</p>
				<h1>ログイン中の開発者たちの<br />3Dモデルを見渡す</h1>
				<p class="lead">
					トップページでは GitHub ログインしているユーザーの活動アバターを表示します。検索は引き続き残し、
					任意の GitHub ユーザーも個別に分析できます。
				</p>
			</div>
		</section>

		<section class="panel form-panel">
			<div class="form-header">
				<div>
					<p class="eyebrow">ユーザー検索</p>
					<h2>任意の GitHub ユーザーを分析</h2>
				</div>
				<p class="form-note">
					ログインしなくても検索できます。ログインしていればヘッダーからいつでもマイページに移動できます。
				</p>
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
				<button on:click={() => analyzeForUsername(username)} disabled={loading}>
					{loading ? '分析中...' : '分析する'}
				</button>
			</div>
		</section>

		<section class="result-card showcase-section">
			<div class="section-header">
				<div>
					<p class="eyebrow">Showcase</p>
					<h2>ログイン中ユーザーのアバター</h2>
				</div>
				<p class="form-note">現在ログインしているユーザーの最新分析結果を並べています。</p>
			</div>

			{#if showcaseDisplayUsers.length > 0}
				<div class="showcase-grid">
					{#await avatarPreviewModulePromise}
						<div class="model-loading">3Dプレビューを準備しています...</div>
					{:then module}
						{#each showcaseDisplayUsers as user}
							<article class="showcase-card">
								<div class="showcase-head">
									<div class="showcase-identity">
										{#if user.profile.avatarUrl}
											<img class="avatar small-avatar" src={user.profile.avatarUrl} alt={user.username} />
										{/if}
										<div>
											<h3>@{user.username}</h3>
											<p>{user.personaTitle}</p>
										</div>
									</div>
									<span class="showcase-pill">{formatActivityLabel(user.stats.dominantActivity)}</span>
								</div>

								<div class="showcase-stage">
									<svelte:component this={module.default} spec={buildAvatarSpec(user)} />
								</div>

								<div class="showcase-metrics">
									<span>総数 {formatNumber(user.stats.totalContributions)}</span>
									<span>連続 {formatNumber(user.stats.longestStreak)}日</span>
									<span>主要 {user.personaTitle}</span>
								</div>
							</article>
						{/each}
					{/await}
				</div>
			{:else}
				<p class="body-copy">まだショーケースに表示できる他ユーザーがいません。先頭で GitHub ログインするとここに参加できます。</p>
			{/if}
		</section>

		{#if searchResult}
			<section class="result-card hero-result">
				<div class="profile-block">
					{#if searchResult.profile.avatarUrl}
						<img class="avatar" src={searchResult.profile.avatarUrl} alt={searchResult.username} />
					{/if}

					<div class="profile-copy">
						<p class="eyebrow">Search Result</p>
						<h2>{searchResult.personaTitle}</h2>
						<p class="summary">{searchResult.summary}</p>

						<div class="profile-meta">
							<span>@{searchResult.username}</span>
							{#if searchResult.profile.name}
								<span>{searchResult.profile.name}</span>
							{/if}
							{#if searchResult.profile.url}
								<a href={searchResult.profile.url} target="_blank" rel="noreferrer">GitHub Profile</a>
							{/if}
						</div>
					</div>
				</div>

				<div class="pill-row">
					<div class="pill">
						<span>おすすめアセット</span>
						<strong>{searchResult.recommendedAsset}</strong>
					</div>
					<div class="pill">
						<span>主要アクティビティ</span>
						<strong>{formatActivityLabel(searchResult.stats.dominantActivity)}</strong>
					</div>
					<div class="pill">
						<span>分析期間</span>
						<strong>{searchResult.stats.from} - {searchResult.stats.to}</strong>
					</div>
				</div>
			</section>
		{/if}
	{:else}
		<section class="hero mypage-hero">
			<div class="hero-copy">
				<p class="eyebrow">My Page</p>
				<h1>自分の活動ログと<br />アバターを確認する</h1>
				<p class="lead">
					ヘッダーから GitHub ログインすると、このページで自分の分析結果と 3D モデルをいつでも確認できます。
				</p>
			</div>
		</section>

		{#if currentUser}
			<section class="result-card hero-result">
				<div class="profile-block">
					{#if currentUser.profile.avatarUrl}
						<img class="avatar" src={currentUser.profile.avatarUrl} alt={currentUser.username} />
					{/if}

					<div class="profile-copy">
						<p class="eyebrow">My Profile</p>
						<h2>{currentUser.personaTitle}</h2>
						<p class="summary">{currentUser.summary}</p>

						<div class="profile-meta">
							<span>@{currentUser.username}</span>
							{#if currentUser.profile.name}
								<span>{currentUser.profile.name}</span>
							{/if}
							{#if currentUser.profile.url}
								<a href={currentUser.profile.url} target="_blank" rel="noreferrer">GitHub Profile</a>
							{/if}
						</div>
					</div>
				</div>

				<div class="pill-row">
					<div class="pill">
						<span>おすすめアセット</span>
						<strong>{currentUser.recommendedAsset}</strong>
					</div>
					<div class="pill">
						<span>主要アクティビティ</span>
						<strong>{formatActivityLabel(currentUser.stats.dominantActivity)}</strong>
					</div>
					<div class="pill">
						<span>分析期間</span>
						<strong>{currentUser.stats.from} - {currentUser.stats.to}</strong>
					</div>
				</div>
			</section>

			<section class="stats-grid">
				{#each getPrimaryStats(currentUser.stats) as stat}
					<article class="stat-card">
						<p>{stat.label}</p>
						<h3>{stat.value}</h3>
					</article>
				{/each}
			</section>

			<section class="result-card model-section">
				<div class="model-copy">
					<p class="eyebrow">コード生成3Dモデル</p>
					<h2>自分の活動から組み上がるアバター</h2>
					<p class="body-copy">
						活動量・streak・主要活動・主要リポジトリ数から、あなたのアバター形状と色をコードで組み立てています。
					</p>

					<div class="model-parameters">
						{#each getAvatarParameters(buildAvatarSpec(currentUser), { ...currentUser.stats, topRepositoryCount: currentUser.topRepositories.length }) as parameter}
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
					{#await avatarPreviewModulePromise}
						<div class="model-loading">3Dプレビューを準備しています...</div>
					{:then module}
						<svelte:component this={module.default} spec={buildAvatarSpec(currentUser)} />
					{/await}
				</div>
			</section>

			<section class="detail-grid">
				<article class="result-card">
					<p class="eyebrow">特徴</p>
					<ul class="trait-list">
						{#each currentUser.traits as trait}
							<li>{trait}</li>
						{/each}
					</ul>
				</article>

				<article class="result-card">
					<p class="eyebrow">ビジュアル方針</p>
					<p class="body-copy">{currentUser.visualDirection}</p>
					<div class="signal-block">
						<h3>活動シグナル</h3>
						<p>{currentUser.contributionSignal}</p>
					</div>
				</article>

				<article class="result-card">
					<p class="eyebrow">活動内訳</p>
					<div class="activity-list">
						<div>
							<span>コミット</span>
							<strong>{formatNumber(currentUser.stats.commitCount)}</strong>
						</div>
						<div>
							<span>プルリクエスト</span>
							<strong>{formatNumber(currentUser.stats.pullRequestCount)}</strong>
						</div>
						<div>
							<span>Issue</span>
							<strong>{formatNumber(currentUser.stats.issueCount)}</strong>
						</div>
						<div>
							<span>レビュー</span>
							<strong>{formatNumber(currentUser.stats.reviewCount)}</strong>
						</div>
						<div>
							<span>最も動いた曜日</span>
							<strong>{currentUser.stats.busiestWeekday}</strong>
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

				{#if currentUser.topRepositories.length > 0}
					<div class="repository-list">
						{#each currentUser.topRepositories as repository}
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
		{:else}
			<section class="panel empty-panel">
				<p class="eyebrow">Login Required</p>
				<h2>まだログインしていません</h2>
				<p class="body-copy">
					ヘッダーの GitHub ログインからサインインすると、自分の活動分析とアバターをこのページで確認できます。
				</p>
				<button class="oauth-button" on:click={loginWithGitHub} disabled={authLoading || !oauthAvailable}>
					{authLoading ? 'GitHubへ移動中...' : 'GitHubでログイン'}
				</button>
			</section>
		{/if}
	{/if}
</main>
