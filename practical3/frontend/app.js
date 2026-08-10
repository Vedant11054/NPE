document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const scannerForm = document.getElementById('scanner-form');
    const targetInput = document.getElementById('target-input');
    const portsInput = document.getElementById('ports-input');
    const workersInput = document.getElementById('workers-input');
    const workersVal = document.getElementById('workers-val');
    const timeoutInput = document.getElementById('timeout-input');
    const timeoutVal = document.getElementById('timeout-val');
    
    const startBtn = document.getElementById('start-btn');
    const stopBtn = document.getElementById('stop-btn');
    
    const statProgress = document.getElementById('stat-progress');
    const statScannedFraction = document.getElementById('stat-scanned-fraction');
    const progressBarFill = document.getElementById('progress-bar-fill');
    const statOpenPorts = document.getElementById('stat-open-ports');
    const statStatusText = document.getElementById('stat-status-text');
    const statSpeed = document.getElementById('stat-speed');
    const statElapsed = document.getElementById('stat-elapsed');
    const statEta = document.getElementById('stat-eta');
    
    const scanGrid = document.getElementById('scan-grid');
    const resultsTbody = document.getElementById('results-tbody');
    const resultsCount = document.getElementById('results-count');
    const resultsSearch = document.getElementById('results-search');
    
    const exportJsonBtn = document.getElementById('export-json-btn');
    const exportCsvBtn = document.getElementById('export-csv-btn');
    
    const interfacesList = document.getElementById('interfaces-list');
    
    // State Variables
    let pollInterval = null;
    let scanResultsList = []; // Cache for filtering and exporting
    let gridPortMap = new Map(); // Maps IP:Port -> DOM Element (for small scans)
    let totalPortsCount = 0;
    
    // Port Presets mapping
    const presets = {
        common: '21,22,23,25,53,80,110,135,139,143,443,445,993,995,1433,3306,3389,5432,5900,8080',
        web: '80,443,8080,8443,9000',
        db: '1433,1521,3306,5432,6379,27017',
        standard: '1-1024'
    };

    // Initialize Lucide Icons
    lucide.createIcons();

    // Set Default Preset on load
    setPortsPreset('common');

    // Preset buttons event listeners
    document.querySelectorAll('.preset-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const presetType = e.target.getAttribute('data-preset');
            setPortsPreset(presetType);
        });
    });

    function setPortsPreset(presetType) {
        document.querySelectorAll('.preset-btn').forEach(b => b.classList.remove('active'));
        const activeBtn = document.querySelector(`.preset-btn[data-preset="${presetType}"]`);
        if (activeBtn) activeBtn.classList.add('active');
        portsInput.value = presets[presetType] || '';
    }

    // Slider Listeners
    workersInput.addEventListener('input', (e) => {
        workersVal.textContent = e.target.value;
    });
    
    timeoutInput.addEventListener('input', (e) => {
        timeoutVal.textContent = `${e.target.value} ms`;
    });

    // Query local IP interfaces
    fetchLocalInterfaces();

    async function fetchLocalInterfaces() {
        try {
            const response = await fetch('/api/interfaces');
            if (!response.ok) throw new Error('Failed to query interfaces');
            const data = await response.json();
            
            interfacesList.innerHTML = '';
            if (!data || data.length === 0) {
                interfacesList.innerHTML = '<div class="loading-mini">No interfaces found.</div>';
                return;
            }

            data.forEach(iface => {
                const item = document.createElement('div');
                item.className = 'interface-item';
                // Pick the first IP in the interface
                const ip = iface.ips && iface.ips.length > 0 ? iface.ips[0] : '';
                
                item.innerHTML = `
                    <div class="interface-name">
                        ${iface.name} <span>${iface.hardware_addr || 'virtual'}</span>
                    </div>
                    <div class="interface-ip">${ip || 'No IP'}</div>
                `;
                
                if (ip) {
                    // Strip subnet slash for scanning targets
                    const targetIp = ip.split('/')[0];
                    // Also support expanding to subnet CIDR
                    const subnetCIDR = calculateCIDR(targetIp, ip.split('/')[1]);
                    
                    item.addEventListener('click', () => {
                        targetInput.value = subnetCIDR || targetIp;
                        targetInput.focus();
                    });
                }
                
                interfacesList.appendChild(item);
            });
        } catch (error) {
            console.error('Error fetching interfaces:', error);
            interfacesList.innerHTML = '<div class="loading-mini">Could not load interfaces</div>';
        }
    }

    function calculateCIDR(ip, bits) {
        if (!ip || !bits) return null;
        // If it's a typical local address, return /24 for easy subnet scanning
        if (ip.startsWith('192.168.') || ip.startsWith('10.')) {
            const octets = ip.split('.');
            if (octets.length === 4) {
                return `${octets[0]}.${octets[1]}.${octets[2]}.0/24`;
            }
        }
        return `${ip}/${bits}`;
    }

    // Form submission (Start Scan)
    scannerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const target = targetInput.value.trim();
        const ports = portsInput.value.trim();
        const workers = parseInt(workersInput.value);
        const timeout = parseInt(timeoutInput.value);

        if (!target || !ports) return;

        // Reset UI before scan
        resetScanUI();

        try {
            const response = await fetch('/api/scan/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ target, ports, workers, timeout })
            });

            const result = await response.json();
            if (!response.ok) {
                alert(`Error starting scan: ${result.error || 'Unknown error'}`);
                return;
            }

            // Lock controls
            setFormControlsEnabled(false);
            
            // Adjust buttons
            startBtn.classList.add('hidden');
            stopBtn.classList.remove('hidden');
            
            // Initialize Matrix Grid
            initMatrixGrid(result.total_ports);
            totalPortsCount = result.total_ports;
            
            // Start Polling
            statStatusText.textContent = 'Scanning...';
            statStatusText.className = 'stat-subtext status-badge open';
            
            pollInterval = setInterval(pollScanStatus, 250);
        } catch (error) {
            console.error('Error starting scan:', error);
            alert('Failed to connect to scanner backend.');
        }
    });

    // Stop Scan action
    stopBtn.addEventListener('click', async () => {
        try {
            stopBtn.disabled = true;
            await fetch('/api/scan/stop', { method: 'POST' });
        } catch (error) {
            console.error('Error stopping scan:', error);
        } finally {
            stopBtn.disabled = false;
        }
    });

    // Reset UI fields
    function resetScanUI() {
        statProgress.textContent = '0%';
        progressBarFill.style.width = '0%';
        statScannedFraction.textContent = '0 / 0 ports';
        statOpenPorts.textContent = '0';
        statSpeed.innerHTML = `0.0<span class="unit">p/s</span>`;
        statElapsed.textContent = 'Elapsed: 0s';
        statEta.textContent = 'Calculating...';
        
        resultsTbody.innerHTML = `
            <tr class="table-empty">
                <td colspan="7">
                    <div class="empty-state">
                        <i data-lucide="loader" class="pulse"></i>
                        <p>Initializing scan grid...</p>
                    </div>
                </td>
            </tr>
        `;
        lucide.createIcons();

        exportJsonBtn.disabled = true;
        exportCsvBtn.disabled = true;
        scanResultsList = [];
        gridPortMap.clear();
        resultsCount.textContent = '0 targets found';
        totalPortsCount = 0;
    }

    function setFormControlsEnabled(enabled) {
        targetInput.disabled = !enabled;
        portsInput.disabled = !enabled;
        workersInput.disabled = !enabled;
        timeoutInput.disabled = !enabled;
        document.querySelectorAll('.preset-btn').forEach(btn => btn.disabled = !enabled);
    }

    // Initialize the Matrix Grid based on volume of ports
    function initMatrixGrid(total) {
        scanGrid.innerHTML = '';
        
        if (total > 800) {
            // For large scans, render a rolling log of activity instead of individual boxes
            // to avoid rendering 1000s of DOM nodes that degrades browser performance.
            scanGrid.innerHTML = `
                <div class="grid-placeholder">
                    <i data-lucide="activity" class="placeholder-icon pulse"></i>
                    <p>Live Activity Feed Activated</p>
                    <span>Scan is too large (${total} ports) for full matrix. Showing rolling inspected ports below.</span>
                </div>
            `;
            lucide.createIcons();
            return;
        }

        // Small scan, render full representation of ports
        // Add empty placeholders
        for (let i = 0; i < total; i++) {
            const cell = document.createElement('div');
            cell.className = 'grid-cell';
            cell.setAttribute('data-index', i);
            cell.setAttribute('data-tooltip', `Port: Pending`);
            scanGrid.appendChild(cell);
        }
    }

    // Poll the status API
    async function pollScanStatus() {
        try {
            const response = await fetch('/api/scan/status');
            if (!response.ok) throw new Error('Status poll failed');
            const data = await response.json();

            // Update Statistics
            statProgress.textContent = `${data.progress.toFixed(1)}%`;
            progressBarFill.style.width = `${data.progress}%`;
            statScannedFraction.textContent = `${data.scanned_ports} / ${data.total_ports} ports`;
            statOpenPorts.textContent = data.open_ports ? data.open_ports.length : 0;
            statSpeed.innerHTML = `${data.scan_speed.toFixed(1)}<span class="unit">p/s</span>`;
            statElapsed.textContent = `Elapsed: ${data.scan_time_elapsed}`;
            statEta.textContent = data.eta;

            // Render live progress updates to Matrix Grid
            if (data.live_scanned && data.live_scanned.length > 0) {
                updateGrid(data.live_scanned);
            }

            // Render Scan Results list
            if (data.open_ports) {
                scanResultsList = data.open_ports;
                renderResultsTable(resultsSearch.value.trim());
            }

            // Handle scan completion / stopping
            if (!data.is_running) {
                clearInterval(pollInterval);
                pollInterval = null;
                
                // Restore UI controls
                setFormControlsEnabled(true);
                startBtn.classList.remove('hidden');
                stopBtn.classList.add('hidden');

                // Determine final state
                if (data.scanned_ports === data.total_ports) {
                    statStatusText.textContent = 'Finished';
                    statStatusText.className = 'stat-subtext status-badge';
                    statEta.textContent = 'Completed';
                } else {
                    statStatusText.textContent = 'Stopped';
                    statStatusText.className = 'stat-subtext status-badge closed';
                    statEta.textContent = 'Cancelled';
                }

                // Enable exports if we have results
                if (scanResultsList.length > 0) {
                    exportJsonBtn.disabled = false;
                    exportCsvBtn.disabled = false;
                }
            }
        } catch (error) {
            console.error('Error polling scan status:', error);
            // Don't clear interval immediately in case of transient network glitch
        }
    }

    // Update Matrix cells or Activity Feed
    function updateGrid(liveScannedPorts) {
        if (totalPortsCount > 800) {
            // Activity log scrolling view
            // We append scanning dots and remove old ones
            let placeholder = scanGrid.querySelector('.grid-placeholder');
            
            // Create a small container if not exists
            let rollingContainer = scanGrid.querySelector('.rolling-container');
            if (!rollingContainer) {
                rollingContainer = document.createElement('div');
                rollingContainer.className = 'scan-grid';
                rollingContainer.style.width = '100%';
                scanGrid.appendChild(rollingContainer);
            }

            liveScannedPorts.forEach(portData => {
                const cell = document.createElement('div');
                cell.className = `grid-cell`;
                
                if (portData.state === 'open') cell.classList.add('scanned-open');
                else if (portData.state === 'filtered') cell.classList.add('scanned-filtered');
                else cell.classList.add('scanned-closed');

                cell.setAttribute('data-tooltip', `${portData.ip}:${portData.port} (${portData.state})`);

                rollingContainer.appendChild(cell);

                // Cap rolling view nodes to 200 elements to preserve performance
                while (rollingContainer.children.length > 200) {
                    rollingContainer.removeChild(rollingContainer.firstChild);
                }
            });
        } else {
            // Static Matrix Grid Mapping
            liveScannedPorts.forEach(portData => {
                const key = `${portData.ip}:${portData.port}`;
                
                // Find first cell that is not yet assigned/scanned
                let cell = gridPortMap.get(key);
                
                if (!cell) {
                    // Get next unassigned cell element
                    const index = gridPortMap.size;
                    cell = scanGrid.children[index];
                    if (cell) {
                        gridPortMap.set(key, cell);
                    }
                }

                if (cell) {
                    cell.className = 'grid-cell'; // Reset scanning state
                    
                    if (portData.state === 'open') {
                        cell.classList.add('scanned-open');
                    } else if (portData.state === 'filtered') {
                        cell.classList.add('scanned-filtered');
                    } else {
                        cell.classList.add('scanned-closed');
                    }

                    cell.setAttribute('data-tooltip', `${portData.ip}:${portData.port} - ${portData.state.toUpperCase()}`);
                }
            });
        }
    }

    // Render results table with search filter
    function renderResultsTable(filterQuery = '') {
        const query = filterQuery.toLowerCase();
        const filtered = scanResultsList.filter(item => {
            return item.ip.includes(query) || 
                   item.port.toString().includes(query) || 
                   item.service.toLowerCase().includes(query) ||
                   (item.banner && item.banner.toLowerCase().includes(query));
        });

        resultsCount.textContent = `${filtered.length} target${filtered.length === 1 ? '' : 's'} found`;

        if (filtered.length === 0) {
            resultsTbody.innerHTML = `
                <tr class="table-empty">
                    <td colspan="7">
                        <div class="empty-state">
                            <i data-lucide="search-code"></i>
                            <p>No matching open ports</p>
                            <span>Try adjusting your search criteria.</span>
                        </div>
                    </td>
                </tr>
            `;
            lucide.createIcons();
            return;
        }

        resultsTbody.innerHTML = '';
        filtered.forEach(item => {
            const tr = document.createElement('tr');
            
            // Map service names to distinct styles
            let serviceClass = 'service-badge';
            const sName = item.service.toLowerCase();
            if (sName.includes('http') || sName.includes('web') || sName.includes('proxy')) {
                serviceClass += ' web-service';
            } else if (sName.includes('ssh') || sName.includes('rdp') || sName.includes('ftp') || sName.includes('smb')) {
                serviceClass += ' secure-service';
            } else if (sName.includes('sql') || sName.includes('redis') || sName.includes('mongo') || sName.includes('db')) {
                serviceClass += ' db-service';
            }

            // Map actions (e.g. open link if port is 80/443 Web)
            let actionHtml = '-';
            if (item.port === 80 || item.port === 8080) {
                actionHtml = `<a href="http://${item.ip}:${item.port}" target="_blank" class="action-link"><i data-lucide="external-link"></i> Browse</a>`;
            } else if (item.port === 443 || item.port === 8443) {
                actionHtml = `<a href="https://${item.ip}:${item.port}" target="_blank" class="action-link"><i data-lucide="external-link"></i> Secure Browse</a>`;
            } else if (item.port === 3389) {
                actionHtml = `<span class="text-dark">mstsc /v:${item.ip}</span>`;
            }

            const bannerText = item.banner ? `<span class="banner-code" title="${item.banner}">${item.banner}</span>` : '<span class="text-dark">None</span>';

            tr.innerHTML = `
                <td><strong>${item.ip}</strong></td>
                <td><code class="banner-code">${item.port}</code></td>
                <td><span class="status-badge open"><span class="dot dot-open"></span> Open</span></td>
                <td><span class="${serviceClass}">${item.service}</span></td>
                <td>${bannerText}</td>
                <td>${item.timestamp}</td>
                <td>${actionHtml}</td>
            `;

            resultsTbody.appendChild(tr);
        });

        lucide.createIcons();
    }

    // Filter results on search input
    resultsSearch.addEventListener('input', (e) => {
        renderResultsTable(e.target.value.trim());
    });

    // Export Scan Data to JSON
    exportJsonBtn.addEventListener('click', () => {
        if (scanResultsList.length === 0) return;
        
        const dataStr = "data:text/json;charset=utf-8," + encodeURIComponent(JSON.stringify(scanResultsList, null, 4));
        const downloadAnchor = document.createElement('a');
        downloadAnchor.setAttribute("href",     dataStr);
        downloadAnchor.setAttribute("download", `portscan_${targetInput.value.replace(/[^a-zA-Z0-9]/g, '_')}_results.json`);
        document.body.appendChild(downloadAnchor);
        downloadAnchor.click();
        downloadAnchor.remove();
    });

    // Export Scan Data to CSV
    exportCsvBtn.addEventListener('click', () => {
        if (scanResultsList.length === 0) return;
        
        let csvContent = "data:text/csv;charset=utf-8,";
        // Header
        csvContent += "IP Address,Port,Status,Service,Banner,Time Found\n";
        
        scanResultsList.forEach(item => {
            const bannerEscaped = item.banner ? item.banner.replace(/"/g, '""') : '';
            csvContent += `"${item.ip}",${item.port},"Open","${item.service}","${bannerEscaped}","${item.timestamp}"\n`;
        });
        
        const encodedUri = encodeURI(csvContent);
        const downloadAnchor = document.createElement('a');
        downloadAnchor.setAttribute("href", encodedUri);
        downloadAnchor.setAttribute("download", `portscan_${targetInput.value.replace(/[^a-zA-Z0-9]/g, '_')}_results.csv`);
        document.body.appendChild(downloadAnchor);
        downloadAnchor.click();
        downloadAnchor.remove();
    });
});
