<script>
	import { onDestroy, onMount } from 'svelte';
	import * as THREE from 'three';

	export let spec;

	let container;
	let scene;
	let camera;
	let renderer;
	let frameId;
	let resizeObserver;
	let avatarGroup;
	let orbitGroup;
	let dragActive = false;
	let pointerX = 0;
	let pointerY = 0;
	let rotationX = 0.12;
	let rotationY = 0.45;

	onMount(() => {
		scene = new THREE.Scene();
		scene.background = null;

		camera = new THREE.PerspectiveCamera(36, 1, 0.1, 100);
		camera.position.set(0, 2, 9.8);

		renderer = new THREE.WebGLRenderer({
			antialias: true,
			alpha: true
		});
		renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
		renderer.outputColorSpace = THREE.SRGBColorSpace;
		container.appendChild(renderer.domElement);

		const ambientLight = new THREE.AmbientLight(0xfff3df, 1.6);
		scene.add(ambientLight);

		const keyLight = new THREE.DirectionalLight(0xffffff, 2.2);
		keyLight.position.set(4, 6, 5);
		scene.add(keyLight);

		const rimLight = new THREE.DirectionalLight(0x74c9be, 1.2);
		rimLight.position.set(-5, 2, -3);
		scene.add(rimLight);

		const floor = new THREE.Mesh(
			new THREE.CircleGeometry(3.7, 48),
			new THREE.MeshStandardMaterial({
				color: 0xf0e4d2,
				transparent: true,
				opacity: 0.9
			})
		);
		floor.rotation.x = -Math.PI / 2;
		floor.position.y = -2.35;
		scene.add(floor);

		const halo = new THREE.Mesh(
			new THREE.TorusGeometry(2.1, 0.1, 18, 80),
			new THREE.MeshStandardMaterial({
				color: 0xffffff,
				emissive: 0xf7d7a6,
				emissiveIntensity: 0.45,
				transparent: true,
				opacity: 0.58
			})
		);
		halo.rotation.x = Math.PI / 2;
		halo.position.y = -2.28;
		scene.add(halo);

		orbitGroup = new THREE.Group();
		scene.add(orbitGroup);

		updateAvatar();
		handleResize();

		resizeObserver = new ResizeObserver(handleResize);
		resizeObserver.observe(container);

		animate();

		return () => cleanup();
	});

	onDestroy(() => {
		cleanup();
	});

	$: if (scene && spec) {
		updateAvatar();
	}

	function cleanup() {
		if (frameId) {
			cancelAnimationFrame(frameId);
		}

		resizeObserver?.disconnect();
		disposeGroup(avatarGroup);
		renderer?.dispose();
		renderer?.domElement?.remove();
	}

	function handleResize() {
		if (!container || !renderer || !camera) {
			return;
		}

		const { width, height } = container.getBoundingClientRect();
		if (!width || !height) {
			return;
		}

		renderer.setSize(width, height, false);
		camera.aspect = width / height;
		camera.updateProjectionMatrix();
	}

	function animate(time = 0) {
		frameId = requestAnimationFrame(animate);

		if (orbitGroup) {
			orbitGroup.rotation.y += 0.005;
			orbitGroup.rotation.x += (rotationX - orbitGroup.rotation.x) * 0.05;
			orbitGroup.rotation.y += (rotationY - orbitGroup.rotation.y) * 0.05;
			orbitGroup.position.y = Math.sin(time * 0.0018) * 0.08;
		}

		renderer.render(scene, camera);
	}

	function updateAvatar() {
		if (!orbitGroup || !spec) {
			return;
		}

		if (avatarGroup) {
			orbitGroup.remove(avatarGroup);
			disposeGroup(avatarGroup);
		}

		avatarGroup = buildAvatar(spec);
		orbitGroup.add(avatarGroup);
	}

	function buildAvatar(input) {
		const group = new THREE.Group();

		const primaryMaterial = createMaterial(input.palette.primary);
		const secondaryMaterial = createMaterial(input.palette.secondary);
		const accentMaterial = createMaterial(input.palette.accent, 0.25);
		const darkMaterial = createMaterial('#36291f');

		const torso = new THREE.Mesh(
			new THREE.CapsuleGeometry(input.torsoRadius, input.torsoHeight, 10, 18),
			primaryMaterial
		);
		torso.position.y = 0.2;
		group.add(torso);

		const head = new THREE.Mesh(
			new THREE.SphereGeometry(input.headRadius, 24, 24),
			secondaryMaterial
		);
		head.position.y = input.torsoHeight * 0.85 + input.headRadius * 1.45;
		group.add(head);

		const visor = new THREE.Mesh(
			new THREE.BoxGeometry(input.headRadius * 1.3, input.headRadius * 0.5, input.headRadius * 1.25),
			accentMaterial
		);
		visor.position.set(0, head.position.y + 0.05, input.headRadius * 0.42);
		group.add(visor);

		const eyeGeometry = new THREE.SphereGeometry(input.headRadius * 0.08, 10, 10);
		[-1, 1].forEach((side) => {
			const eye = new THREE.Mesh(eyeGeometry, darkMaterial);
			eye.position.set(side * input.headRadius * 0.34, head.position.y + 0.07, input.headRadius * 0.92);
			group.add(eye);
		});

		const limbGeometry = new THREE.CapsuleGeometry(input.limbRadius, input.limbLength, 8, 14);
		[-1, 1].forEach((side) => {
			const arm = new THREE.Mesh(limbGeometry, secondaryMaterial);
			arm.position.set(side * (input.torsoRadius + input.limbRadius * 1.6), 0.52, 0);
			arm.rotation.z = side * 0.18;
			group.add(arm);

			const leg = new THREE.Mesh(
				new THREE.CapsuleGeometry(input.legRadius, input.legLength, 8, 14),
				secondaryMaterial
			);
			leg.position.set(side * input.torsoRadius * 0.45, -1.5, 0);
			group.add(leg);
		});

		for (let i = 0; i < input.finCount; i += 1) {
			const angle = ((Math.PI * 2) / input.finCount) * i;
			const fin = new THREE.Mesh(
				new THREE.BoxGeometry(0.18, input.finHeight, 0.65),
				accentMaterial
			);
			fin.position.set(Math.sin(angle) * 0.78, 0.55, Math.cos(angle) * 0.78);
			fin.lookAt(fin.position.clone().multiplyScalar(2));
			group.add(fin);
		}

		for (let i = 0; i < input.antennaCount; i += 1) {
			const offset = (i - (input.antennaCount - 1) / 2) * 0.34;
			const antenna = new THREE.Mesh(
				new THREE.CylinderGeometry(0.05, 0.05, input.antennaHeight, 10),
				darkMaterial
			);
			antenna.position.set(offset, head.position.y + input.headRadius + input.antennaHeight * 0.48, 0);
			group.add(antenna);

			const orb = new THREE.Mesh(
				new THREE.SphereGeometry(0.13, 12, 12),
				accentMaterial
			);
			orb.position.set(offset, antenna.position.y + input.antennaHeight * 0.58, 0);
			group.add(orb);
		}

		for (let i = 0; i < input.satelliteCount; i += 1) {
			const angle = ((Math.PI * 2) / input.satelliteCount) * i + 0.35;
			const satellite = new THREE.Mesh(
				new THREE.IcosahedronGeometry(0.17 + input.weekendBias * 0.1, 0),
				accentMaterial
			);
			satellite.position.set(Math.sin(angle) * 1.8, head.position.y + 0.18, Math.cos(angle) * 1.8);
			group.add(satellite);
		}

		for (let i = 0; i < input.badgeCount; i += 1) {
			const badge = new THREE.Mesh(
				new THREE.BoxGeometry(0.16, 0.16, 0.06),
				accentMaterial
			);
			badge.position.set((i - (input.badgeCount - 1) / 2) * 0.26, 0.15 - i * 0.08, input.torsoRadius + 0.26);
			group.add(badge);
		}

		group.position.y = 0.15;

		return group;
	}

	function createMaterial(color, emissiveIntensity = 0.08) {
		return new THREE.MeshStandardMaterial({
			color,
			roughness: 0.48,
			metalness: 0.14,
			emissive: new THREE.Color(color).multiplyScalar(emissiveIntensity)
		});
	}

	function disposeGroup(group) {
		if (!group) {
			return;
		}

		group.traverse((node) => {
			if (node.geometry) {
				node.geometry.dispose();
			}

			if (node.material) {
				if (Array.isArray(node.material)) {
					node.material.forEach((material) => material.dispose());
				} else {
					node.material.dispose();
				}
			}
		});
	}

	function handlePointerDown(event) {
		dragActive = true;
		pointerX = event.clientX;
		pointerY = event.clientY;
	}

	function handlePointerMove(event) {
		if (!dragActive) {
			return;
		}

		const deltaX = event.clientX - pointerX;
		const deltaY = event.clientY - pointerY;
		pointerX = event.clientX;
		pointerY = event.clientY;

		rotationY += deltaX * 0.008;
		rotationX = THREE.MathUtils.clamp(rotationX + deltaY * 0.004, -0.25, 0.45);
	}

	function handlePointerUp() {
		dragActive = false;
	}
</script>

<div
	class="preview-shell"
	bind:this={container}
	role="img"
	aria-label="分析結果から生成された3Dアバタープレビュー"
	on:pointerdown={handlePointerDown}
	on:pointermove={handlePointerMove}
	on:pointerup={handlePointerUp}
	on:pointerleave={handlePointerUp}
>
	<div class="preview-overlay">
		<span>ドラッグで角度変更</span>
		<span>自動回転中</span>
	</div>
</div>

<style>
	.preview-shell {
		position: relative;
		width: 100%;
		height: min(440px, 62vw);
		min-height: 320px;
		border-radius: 28px;
		overflow: hidden;
		background:
			radial-gradient(circle at top, rgba(255, 255, 255, 0.84), transparent 38%),
			linear-gradient(180deg, rgba(212, 240, 234, 0.52), rgba(245, 236, 225, 0.82));
		border: 1px solid rgba(31, 26, 21, 0.08);
		cursor: grab;
	}

	.preview-shell:active {
		cursor: grabbing;
	}

	.preview-overlay {
		position: absolute;
		left: 16px;
		right: 16px;
		bottom: 16px;
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
		pointer-events: none;
	}

	.preview-overlay span {
		padding: 8px 10px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.72);
		border: 1px solid rgba(31, 26, 21, 0.08);
		color: #61564a;
		font-size: 0.82rem;
		letter-spacing: 0.04em;
	}

	:global(.preview-shell canvas) {
		display: block;
		width: 100%;
		height: 100%;
	}

	@media (max-width: 720px) {
		.preview-shell {
			height: 360px;
			min-height: 280px;
		}
	}
</style>
