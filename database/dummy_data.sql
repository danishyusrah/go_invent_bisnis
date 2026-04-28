USE inventory_db;

INSERT INTO products (part_number, description, quantity, min_stock_level, capital_price, location) VALUES
('CAP-100UF-25V', 'Capacitor Electrolytic 100uF 25V', 500, 100, 500.00, 'Rak A1'),
('RES-10K-0.25W', 'Resistor 10K Ohm 1/4W 1%', 2000, 500, 50.00, 'Rak A2'),
('IC-NE555', 'Timer IC NE555', 150, 50, 2500.00, 'Rak B1'),
('TR-2N3904', 'Transistor NPN 2N3904', 300, 100, 400.00, 'Rak B2'),
('LED-RED-5MM', 'LED 5mm Red', 0, 200, 300.00, 'Rak C1'),
('DIODE-1N4148', 'Switching Diode 1N4148', 15, 50, 150.00, 'Rak C2'),
('PCB-PROTO', 'PCB Prototyping Board 5x7cm', 40, 20, 5000.00, 'Rak D1'),
('CONN-HDR-40P', 'Pin Header 40 Pin Male', 80, 50, 1500.00, 'Rak D2'),
('SW-TACT-12MM', 'Tactile Switch 12x12mm', 0, 100, 800.00, 'Rak E1'),
('IC-LM317', 'Voltage Regulator LM317', 25, 30, 3500.00, 'Rak E2');
